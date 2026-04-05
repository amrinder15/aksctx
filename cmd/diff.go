package cmd

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/amrinder15/aksctx/internal/azure"
	"github.com/amrinder15/aksctx/internal/tui"
	"github.com/spf13/cobra"
)

type diffRow struct {
	Category string
	Field    string
	Left     string
	Right    string
	Status   string
}

var diffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Compare two AKS clusters discovered from Azure",
	Long: `Compare exactly two AKS clusters using Azure discovery only.

	aksctx diff first shows your accessible Azure subscriptions, lets you pick
	the left-side cluster from one subscription, then repeats the flow to choose
	the right-side cluster.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			return fmt.Errorf("diff does not accept positional arguments; run aksctx diff and choose both clusters interactively")
		}
		return nil
	},
	RunE: runDiff,
}

func runDiff(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	out := cmd.OutOrStdout()

	cred, err := azure.NewCredential()
	if err != nil {
		return err
	}
	if strings.TrimSpace(subscriptionName) != "" {
		return fmt.Errorf("--subscription is not supported for diff; choose subscriptions interactively")
	}

	fmt.Fprintln(out, "🔍 Loading Azure subscriptions for AKS comparison...")

	left, cancelled, err := promptForDiffTarget(ctx, cred, "left", nil)
	if err != nil {
		return err
	}
	if cancelled {
		fmt.Fprintln(out, "Cancelled.")
		return nil
	}

	right, cancelled, err := promptForDiffTarget(ctx, cred, "right", &left)
	if err != nil {
		return err
	}
	if cancelled {
		fmt.Fprintln(out, "Cancelled.")
		return nil
	}

	rows := buildDiffRows(left, right)
	differences, missing, major := summarizeDiffRows(rows)

	fmt.Fprintln(out)
	fmt.Fprintf(out, "Comparing %s and %s\n", left.Name, right.Name)
	fmt.Fprintf(out, "Left         : %s / %s / %s\n", left.SubscriptionName, left.ResourceGroup, left.Location)
	fmt.Fprintf(out, "Right        : %s / %s / %s\n", right.SubscriptionName, right.ResourceGroup, right.Location)
	fmt.Fprintf(out, "Differences  : %d/%d fields\n", differences, len(rows))
	fmt.Fprintf(out, "Missing      : %d fields\n", missing)
	if len(major) == 0 {
		fmt.Fprintln(out, "Major drift  : none")
	} else {
		fmt.Fprintf(out, "Major drift  : %s\n", strings.Join(major, ", "))
	}
	fmt.Fprintln(out)

	writeDiffTable(out, left, right, rows)
	return nil
}

func promptForDiffTarget(ctx context.Context, cred azcore.TokenCredential, side string, excluded *azure.Cluster) (azure.Cluster, bool, error) {
	subscriptions, err := azure.ListSubscriptions(ctx, cred, "")
	if err != nil {
		return azure.Cluster{}, false, err
	}
	if len(subscriptions) == 0 {
		return azure.Cluster{}, false, fmt.Errorf("no Azure subscriptions available")
	}

	selectedSubscription, cancelled, err := promptForSubscriptionSelection(side, subscriptions)
	if err != nil || cancelled {
		return azure.Cluster{}, cancelled, err
	}

	clusters, err := azure.ListClustersInSubscription(ctx, cred, selectedSubscription)
	if err != nil {
		return azure.Cluster{}, false, fmt.Errorf("listing clusters in subscription %q: %w", selectedSubscription.Name, err)
	}

	availableClusters := filterClustersForDiff(clusters, excluded)
	if len(availableClusters) == 0 {
		if excluded != nil && excluded.SubscriptionID == selectedSubscription.ID {
			return azure.Cluster{}, false, fmt.Errorf("no additional AKS clusters available in subscription %q", selectedSubscription.Name)
		}
		return azure.Cluster{}, false, fmt.Errorf("no AKS clusters found in subscription %q", selectedSubscription.Name)
	}

	return promptForClusterSelection(fmt.Sprintf("Select the %s AKS cluster", side), availableClusters)
}
func promptForSubscriptionSelection(side string, subscriptions []azure.Subscription) (azure.Subscription, bool, error) {
	items := make([]tui.Item, len(subscriptions))
	for i, subscription := range subscriptions {
		items[i] = tui.Item{
			Label:       subscription.Name,
			Description: subscription.ID,
			Value:       subscription,
		}
	}

	result, err := tui.Run(fmt.Sprintf("Select the %s subscription", side), items)
	if err != nil {
		return azure.Subscription{}, false, err
	}
	if result.Aborted || result.Selected == nil {
		return azure.Subscription{}, true, nil
	}

	return result.Selected.Value.(azure.Subscription), false, nil
}

func promptForClusterSelection(title string, clusters []azure.Cluster) (azure.Cluster, bool, error) {
	if len(clusters) == 0 {
		return azure.Cluster{}, false, fmt.Errorf("no AKS clusters available for selection")
	}

	items := make([]tui.Item, len(clusters))
	for i, cluster := range clusters {
		items[i] = tui.Item{
			Label:       cluster.Name,
			Description: clusterDescription(cluster),
			Value:       cluster,
		}
	}

	result, err := tui.Run(title, items)
	if err != nil {
		return azure.Cluster{}, false, err
	}
	if result.Aborted || result.Selected == nil {
		return azure.Cluster{}, true, nil
	}

	return result.Selected.Value.(azure.Cluster), false, nil
}

func clusterDescription(cluster azure.Cluster) string {
	return fmt.Sprintf("%s / %s / %s / k8s %s / %d nodes", cluster.SubscriptionName, cluster.Location, cluster.ResourceGroup, cluster.K8sVersion, cluster.NodeCount)
}

func filterClustersForDiff(clusters []azure.Cluster, excluded *azure.Cluster) []azure.Cluster {
	filtered := make([]azure.Cluster, 0, len(clusters))
	for _, cluster := range clusters {
		if excluded != nil && cluster.ID == excluded.ID {
			continue
		}
		filtered = append(filtered, cluster)
	}
	return filtered
}

func buildDiffRows(left, right azure.Cluster) []diffRow {
	rows := []diffRow{
		compareRow("identity", "Cluster name", left.Name, right.Name),
		compareRow("identity", "Subscription", left.SubscriptionName, right.SubscriptionName),
		compareRow("identity", "Subscription ID", left.SubscriptionID, right.SubscriptionID),
		compareRow("identity", "Resource group", left.ResourceGroup, right.ResourceGroup),
		compareRow("identity", "Location", left.Location, right.Location),
		compareRow("identity", "Resource ID", left.ID, right.ID),
		compareRow("platform", "Kubernetes version", left.K8sVersion, right.K8sVersion),
		compareRow("platform", "AKS SKU tier", left.SKUTier, right.SKUTier),
		compareRow("platform", "Support plan", left.SupportPlan, right.SupportPlan),
		compareRow("platform", "Provisioning state", left.ProvisioningState, right.ProvisioningState),
		compareRow("platform", "Power state", left.PowerState, right.PowerState),
		compareRow("access", "Kubernetes RBAC", formatEnabled(left.EnableRBAC), formatEnabled(right.EnableRBAC)),
		compareRow("access", "Azure RBAC", formatEnabled(left.EnableAzureRBAC), formatEnabled(right.EnableAzureRBAC)),
		compareRow("access", "Local accounts", formatEnabled(!left.DisableLocalAccounts), formatEnabled(!right.DisableLocalAccounts)),
		compareRow("access", "Private cluster", formatEnabled(left.PrivateCluster), formatEnabled(right.PrivateCluster)),
		compareRow("access", "Authorized IP ranges", formatCount(left.AuthorizedIPRanges), formatCount(right.AuthorizedIPRanges)),
		compareRow("network", "DNS prefix", left.DNSPrefix, right.DNSPrefix),
		compareRow("network", "Network plugin", left.NetworkPlugin, right.NetworkPlugin),
		compareRow("network", "Network policy", left.NetworkPolicy, right.NetworkPolicy),
		compareRow("network", "Network dataplane", left.NetworkDataplane, right.NetworkDataplane),
		compareRow("network", "Network mode", left.NetworkMode, right.NetworkMode),
		compareRow("network", "Outbound type", left.OutboundType, right.OutboundType),
		compareRow("addons", "Monitoring addon", formatEnabled(left.MonitoringAddonEnabled), formatEnabled(right.MonitoringAddonEnabled)),
		compareRow("addons", "Azure Policy addon", formatEnabled(left.AzurePolicyAddonEnabled), formatEnabled(right.AzurePolicyAddonEnabled)),
		compareRow("node-pools", "Node pool count", formatInt(len(left.NodePools)), formatInt(len(right.NodePools))),
		compareRow("node-pools", "Total node count", formatInt32(left.NodeCount), formatInt32(right.NodeCount)),
		compareRow("node-pools", "System pool present", formatEnabled(hasSystemPool(left.NodePools)), formatEnabled(hasSystemPool(right.NodePools))),
	}

	rows = append(rows, buildNodePoolRows(left.NodePools, right.NodePools)...)
	return rows
}

func buildNodePoolRows(leftPools, rightPools []azure.AgentPool) []diffRow {
	leftByName := make(map[string]azure.AgentPool, len(leftPools))
	rightByName := make(map[string]azure.AgentPool, len(rightPools))
	names := make(map[string]struct{}, len(leftPools)+len(rightPools))

	for _, pool := range leftPools {
		leftByName[pool.Name] = pool
		names[pool.Name] = struct{}{}
	}
	for _, pool := range rightPools {
		rightByName[pool.Name] = pool
		names[pool.Name] = struct{}{}
	}

	orderedNames := make([]string, 0, len(names))
	for name := range names {
		orderedNames = append(orderedNames, name)
	}
	sort.Strings(orderedNames)

	rows := make([]diffRow, 0, len(orderedNames)*7)
	for _, name := range orderedNames {
		leftPool, leftOK := leftByName[name]
		rightPool, rightOK := rightByName[name]
		prefix := fmt.Sprintf("Node pool %s", name)

		rows = append(rows,
			compareRow("node-pools", prefix+" present", formatPresent(leftOK), formatPresent(rightOK)),
			compareRow("node-pools", prefix+" mode", poolValue(leftPool.Mode, leftOK), poolValue(rightPool.Mode, rightOK)),
			compareRow("node-pools", prefix+" VM size", poolValue(leftPool.VMSize, leftOK), poolValue(rightPool.VMSize, rightOK)),
			compareRow("node-pools", prefix+" node count", poolNumber(leftPool.Count, leftOK), poolNumber(rightPool.Count, rightOK)),
			compareRow("node-pools", prefix+" autoscaling", poolValue(formatEnabled(leftPool.EnableAutoScaling), leftOK), poolValue(formatEnabled(rightPool.EnableAutoScaling), rightOK)),
			compareRow("node-pools", prefix+" min nodes", poolNumber(leftPool.MinCount, leftOK), poolNumber(rightPool.MinCount, rightOK)),
			compareRow("node-pools", prefix+" max nodes", poolNumber(leftPool.MaxCount, leftOK), poolNumber(rightPool.MaxCount, rightOK)),
			compareRow("node-pools", prefix+" OS type", poolValue(leftPool.OSType, leftOK), poolValue(rightPool.OSType, rightOK)),
		)
	}

	return rows
}

func summarizeDiffRows(rows []diffRow) (int, int, []string) {
	differences := 0
	missing := 0
	drift := map[string]bool{}

	for _, row := range rows {
		if row.Status == "same" {
			continue
		}
		differences++
		if strings.Contains(row.Status, "missing") {
			missing++
		}
		if label := driftLabel(row.Category); label != "" {
			drift[label] = true
		}
	}

	ordered := []string{"identity drift", "platform drift", "access drift", "network drift", "addon drift", "node-pool drift"}
	major := make([]string, 0, len(ordered))
	for _, label := range ordered {
		if drift[label] {
			major = append(major, label)
		}
	}

	return differences, missing, major
}

func driftLabel(category string) string {
	switch category {
	case "identity":
		return "identity drift"
	case "platform":
		return "platform drift"
	case "access":
		return "access drift"
	case "network":
		return "network drift"
	case "addons":
		return "addon drift"
	case "node-pools":
		return "node-pool drift"
	default:
		return ""
	}
}

func writeDiffTable(out io.Writer, left, right azure.Cluster, rows []diffRow) {
	leftHeader := "LEFT"
	rightHeader := "RIGHT"
	resultHeader := "RESULT"
	groups := groupDiffRows(rows)
	orderedCategories := []string{"identity", "platform", "access", "network", "addons", "node-pools"}

	for index, category := range orderedCategories {
		categoryRows := groups[category]
		if len(categoryRows) == 0 {
			continue
		}
		if index > 0 {
			fmt.Fprintln(out)
		}

		fmt.Fprintf(out, "%s\n", categoryTitle(category))

		fieldWidth := len("FIELD")
		leftWidth := len(leftHeader)
		rightWidth := len(rightHeader)
		resultWidth := len(resultHeader)
		for _, row := range categoryRows {
			fieldWidth = maxInt(fieldWidth, len(compactField(row.Field)))
			leftWidth = maxInt(leftWidth, len(compactValue(row.Left)))
			rightWidth = maxInt(rightWidth, len(compactValue(row.Right)))
			resultWidth = maxInt(resultWidth, len(renderStatus(row.Status)))
		}

		format := fmt.Sprintf("%%-%ds   %%-%ds   %%-%ds   %%-%ds\n", fieldWidth, leftWidth, rightWidth, resultWidth)
		fmt.Fprintf(out, format, "FIELD", leftHeader, rightHeader, resultHeader)
		fmt.Fprintf(
			out,
			format,
			strings.Repeat("-", fieldWidth),
			strings.Repeat("-", leftWidth),
			strings.Repeat("-", rightWidth),
			strings.Repeat("-", resultWidth),
		)
		for _, row := range categoryRows {
			fmt.Fprintf(out, format, compactField(row.Field), compactValue(row.Left), compactValue(row.Right), renderStatus(row.Status))
		}
	}
}

func maxInt(left, right int) int {
	if left > right {
		return left
	}

	return right
}

func compactField(value string) string {
	return truncateValue(value, 32)
}

func compactValue(value string) string {
	return truncateValue(displayValue(value), 42)
}

func truncateValue(value string, maxWidth int) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "-"
	}
	if maxWidth <= 1 || len(trimmed) <= maxWidth {
		return trimmed
	}

	return trimmed[:maxWidth-1] + "…"
}

func groupDiffRows(rows []diffRow) map[string][]diffRow {
	groups := make(map[string][]diffRow, len(rows))
	for _, row := range rows {
		groups[row.Category] = append(groups[row.Category], row)
	}
	return groups
}

func categoryTitle(category string) string {
	switch category {
	case "identity":
		return "IDENTITY"
	case "platform":
		return "PLATFORM"
	case "access":
		return "ACCESS"
	case "network":
		return "NETWORK"
	case "addons":
		return "ADD-ONS"
	case "node-pools":
		return "NODE POOLS"
	default:
		return strings.ToUpper(category)
	}
}

func renderStatus(status string) string {
	switch status {
	case "same":
		return "MATCH"
	case "different":
		return "DIFF"
	case "left missing":
		return "LEFT MISSING"
	case "right missing":
		return "RIGHT MISSING"
	default:
		return strings.ToUpper(status)
	}
}

func compareRow(category, field, left, right string) diffRow {
	return diffRow{
		Category: category,
		Field:    field,
		Left:     strings.TrimSpace(left),
		Right:    strings.TrimSpace(right),
		Status:   diffStatus(strings.TrimSpace(left), strings.TrimSpace(right)),
	}
}

func diffStatus(left, right string) string {
	switch {
	case left == right:
		return "same"
	case left == "" && right != "":
		return "left missing"
	case left != "" && right == "":
		return "right missing"
	default:
		return "different"
	}
}

func displayValue(value string) string {
	if strings.TrimSpace(value) == "" {
		return "-"
	}
	return value
}

func formatEnabled(value bool) string {
	if value {
		return "enabled"
	}
	return "disabled"
}

func formatCount(value int) string {
	if value == 0 {
		return "none"
	}
	return fmt.Sprintf("%d", value)
}

func formatInt(value int) string {
	return fmt.Sprintf("%d", value)
}

func formatInt32(value int32) string {
	return fmt.Sprintf("%d", value)
}

func formatPresent(value bool) string {
	if value {
		return "present"
	}
	return "missing"
}

func poolValue(value string, ok bool) string {
	if !ok {
		return ""
	}
	return value
}

func poolNumber(value int32, ok bool) string {
	if !ok {
		return ""
	}
	return formatInt32(value)
}

func hasSystemPool(pools []azure.AgentPool) bool {
	for _, pool := range pools {
		if strings.EqualFold(pool.Mode, "System") {
			return true
		}
	}
	return false
}
