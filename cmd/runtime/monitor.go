package runtime

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/mulesoft-anypoint/muletracker-cli/anypoint"
	"github.com/mulesoft-anypoint/muletracker-cli/utils"
	"github.com/spf13/cobra"
)

type AppResult struct {
	AppID        string
	AppType      string
	LastCalled   time.Time
	RequestCount int
	Err          error
	LCWindow     string // Last Called window used in the query
	RCWindow     string // Request Count window used in the query
}

// ----- Helper Functions ----- //

// getAppsToMonitor retrieves the list of apps based on the provided flags.
// If a specific appName is provided, it returns a slice with that single app.
// Otherwise, it calls the GetApps method on the client.
func getAppsToMonitor(ctx context.Context, client *anypoint.Client, orgID, envID, appID string, filters ...anypoint.AppFilter) ([]anypoint.App, error) {
	apps, err := client.GetApps(ctx, orgID, envID, filters...)
	if err != nil {
		return nil, fmt.Errorf("error retrieving apps: %v", err)
	}

	if appID != "" {
		return anypoint.FilterApps(apps, anypoint.FilterByName(appID)), nil
	}
	// Otherwise, retrieve all apps.
	return apps, nil
}

// monitorSingleApp retrieves monitoring data for a single app.
func monitorSingleApp(ctx context.Context, client *anypoint.Client, orgID, envID string, app anypoint.App, lcWindow, rcWindow string) AppResult {
	var res AppResult
	// res.AppID = app.ID
	res.AppID = app.Artifact.Name
	res.AppType = app.GetType()
	res.LCWindow = lcWindow
	res.RCWindow = rcWindow

	lastCalled, err1 := client.GetLastCalledTime(ctx, orgID, envID, app, lcWindow)
	reqCount, err2 := client.GetRequestCount(ctx, orgID, envID, app, rcWindow)
	if err1 != nil || err2 != nil {
		res.Err = fmt.Errorf("lastCalled error: %v, requestCount error: %v", err1, err2)
	}
	res.LastCalled = lastCalled
	res.RequestCount = reqCount
	return res
}

// monitorAppsConcurrently monitors a list of apps with concurrency and rate limiting.
func monitorAppsConcurrently(ctx context.Context, client *anypoint.Client, orgID, envID, lcWindow, rcWindow string, apps []anypoint.App) []AppResult {
	const concurrencyLimit = 5
	sem := make(chan struct{}, concurrencyLimit)
	var wg sync.WaitGroup
	resultsCh := make(chan AppResult, len(apps))

	// Create a rate limiter ticker: 10 requests per second.
	rateLimiter := time.NewTicker(100 * time.Millisecond)
	defer rateLimiter.Stop()

	for _, app := range apps {
		wg.Add(1)
		go func(app anypoint.App) {
			defer wg.Done()
			sem <- struct{}{}        // Acquire semaphore.
			defer func() { <-sem }() // Release semaphore.
			<-rateLimiter.C          // Wait for rate limiter tick.
			result := monitorSingleApp(ctx, client, orgID, envID, app, lcWindow, rcWindow)
			resultsCh <- result
		}(app)
	}

	wg.Wait()
	close(resultsCh)

	var results []AppResult
	for r := range resultsCh {
		if r.Err != nil {
			fmt.Fprintf(os.Stderr, "Error monitoring app %s: %v\n", r.AppID, r.Err)
		}
		results = append(results, r)
	}
	return results
}

// filterAppResults applies the filter flag to the full list of results.
// filterFlag can be: "all", "nonempty", or "empty".
func filterAppResults(results []AppResult, filterFlag string) []AppResult {
	var filtered []AppResult
	switch strings.ToLower(filterFlag) {
	case "nonempty":
		for _, r := range results {
			if r.RequestCount > 0 {
				filtered = append(filtered, r)
			}
		}
	case "empty":
		for _, r := range results {
			if r.RequestCount == 0 {
				filtered = append(filtered, r)
			}
		}
	default:
		// "all" or any other value returns all results.
		filtered = results
	}
	return filtered
}

func AppResult2Map(results []AppResult) ([]map[string]any, []string) {
	data := make([]map[string]any, 0)
	for _, r := range results {
		var lastCalled string
		if r.LastCalled.IsZero() {
			lastCalled = "No data"
		} else {
			lastCalled = r.LastCalled.Format(time.RFC1123)
		}
		data = append(data, map[string]any{
			"App ID":        r.AppID,
			"Type":          r.AppType,
			"Last Called":   lastCalled,
			"Request Count": r.RequestCount,
		})
	}
	order := []string{"App ID", "Type", "Request Count", "Last Called"}

	return data, order
}

// printAppsSummaryTable prints a condensed table of app monitoring results
// using tabwriter for alignment.
func printAppsSummaryTable(results []AppResult) {
	data, order := AppResult2Map(results)
	utils.PrintGenericTable(data, order)
}

func ExportAppResultToCSV(fileName string, results []AppResult) error {
	data, order := AppResult2Map(results)
	return utils.ExportGenericCSV(fileName, data, order)
}

// ----- Main Command ----- //

// monitorCmd represents the monitor command
var monitorCmd = &cobra.Command{
	Use:   "monitor",
	Short: "Monitor MuleSoft App Activity",
	Long: `Monitor MuleSoft app activity by retrieving the last-called time
and request count for each app over specified time windows.

If the --app flag is empty, all apps for the given org/env are monitored concurrently.

Filters:
  --filter: "all" (default), "nonempty" (only apps with monitoring data), or "empty" (only apps with no data)
  --app-type: "all" (default), "cloudhub" (only CloudHub apps), or "rtf" (only RTF apps)
`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()

		// Retrieve flag values.
		orgID, _ := cmd.Flags().GetString("org")
		envID, _ := cmd.Flags().GetString("env")
		appID, _ := cmd.Flags().GetString("app")
		adminToken, _ := cmd.Flags().GetString("token")
		lcWindow, _ := cmd.Flags().GetString("last-called-window")
		rcWindow, _ := cmd.Flags().GetString("request-count-window")
		dataFilter, _ := cmd.Flags().GetString("filter")
		appType, _ := cmd.Flags().GetString("app-type")
		exportFile, _ := cmd.Flags().GetString("out")

		// Retrieve the authenticated client.
		client, err := anypoint.GetInitializedClient(adminToken)
		if err != nil {
			utils.PrintError("Error retrieving client %v\n", err)
			return
		}
		// Validate organization and environment.
		orgID, envID, err = utils.ValidateOrgEnv(client, orgID, envID)
		if err != nil {
			utils.PrintError("Validation error: %v", err)
			return
		}

		// Display the client info in a colorful way.
		utils.PrintClientInfo(ctx, client)

		// Build type filters based on app-type flag.
		var typeFilters []anypoint.AppFilter = []anypoint.AppFilter{anypoint.FilterRunning}
		switch strings.ToLower(appType) {
		case "cloudhub":
			typeFilters = append(typeFilters, anypoint.FilterCH1)
		case "rtf":
			typeFilters = append(typeFilters, anypoint.FilterRTF)
		case "all":
			typeFilters = append(typeFilters, anypoint.FilterCH1OrRTF)
		}

		// Retrieve apps to monitor.
		apps, err := getAppsToMonitor(ctx, client, orgID, envID, appID, typeFilters...)
		if err != nil {
			utils.PrintError("Error retrieving apps: %v\n", err)
			return
		}

		if len(apps) == 0 {
			utils.PrintWarning("No apps found for the given org and env.")
			return
		}
		var finalResults []AppResult
		// If a single app was specified, run in single-app mode.
		if appID != "" {
			result := monitorSingleApp(ctx, client, orgID, envID, apps[0], lcWindow, rcWindow)
			utils.PrintInfo("Using last-called window: %s\n", lcWindow)
			utils.PrintInfo("Using request count window: %s\n", rcWindow)
			utils.PrintInfo("Collection monitoring data for %s application only\n", appID)
			if result.Err != nil {
				utils.PrintError("Error monitoring app %s: %v\n", appID, result.Err)
				return
			}
			finalResults = []AppResult{result}
		} else {
			// Monitor all apps concurrently.
			allResults := monitorAppsConcurrently(ctx, client, orgID, envID, lcWindow, rcWindow, apps)
			utils.PrintInfo("Using last-called window: %s\n", lcWindow)
			utils.PrintInfo("Using request count window: %s\n", rcWindow)
			utils.PrintInfo("Found %d apps to monitor.\n", len(apps))
			utils.PrintInfo("Collected monitoring data for %d apps.\n", len(allResults))
			// Apply filter.
			finalResults = filterAppResults(allResults, dataFilter)
			utils.PrintInfo("After applying filter '%s', %d apps remain.\n", dataFilter, len(finalResults))
			if len(finalResults) == 0 {
				utils.PrintWarning("No apps match the filter criteria.")
				return
			}
		}

		// If export flag is provided, export results to CSV.
		if exportFile != "" {
			err := ExportAppResultToCSV(exportFile, finalResults)
			if err != nil {
				utils.PrintError("Error exporting results to CSV: %v\n", err)
				return
			}
			utils.PrintSuccess("Results successfully exported to %s", exportFile)
		} else {
			// Otherwise, print a summary table.
			printAppsSummaryTable(finalResults)
		}

	},
}

func init() {
	// Define flags for organization, environment, and application IDs.
	monitorCmd.Flags().String("org", "", "The Business Group ID. If not provided, the id from the saved context will be loaded if present.")
	monitorCmd.Flags().String("env", "", "The Environment ID. If not provided, the id from the saved context will be loaded if present")
	monitorCmd.Flags().String("app", "", "The Application to monitor (optional)")
	monitorCmd.Flags().StringP("token", "t", "", "The Anypoint Access Token. This token must be the org admin's token in order to have access to all orgs and environments.")

	// Define flags for specifying the time window for queries.
	monitorCmd.Flags().String("last-called-window", "15m", "Time window for last-called query (e.g., 15m, 1h, 24h)")
	monitorCmd.Flags().String("request-count-window", "24h", "Time window for request count query (e.g., 24h, 3d)")

	// Define a flag to filter the results.
	monitorCmd.Flags().String("filter", "all", "Filter results: all (default), nonempty (only apps with monitoring data), or empty (only apps with no data)")
	monitorCmd.Flags().String("app-type", "all", "Filter apps by type: all (default), cloudhub (only CloudHub apps), or rtf (only RTF apps)")

	// export flag
	monitorCmd.Flags().StringP("out", "o", "", "If provided, export the results to the specified CSV file")
}
