package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"text/tabwriter"
	"time"

	"github.com/mulesoft-anypoint/muletracker-cli/anypoint"
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

var includeEmpty bool

// ----- Helper Functions ----- //

// getAppsToMonitor retrieves the list of apps based on the provided flags.
// If a specific appID is provided, it returns a slice with that single app.
// Otherwise, it calls the GetApps method on the client.
func getAppsToMonitor(ctx context.Context, client *anypoint.Client, orgID, envID, appID string, filters ...anypoint.AppFilter) ([]anypoint.App, error) {
	if appID != "" {
		// If an app ID is provided, create a dummy App struct with that ID.
		// (Assuming the monitoring functions use only the app.ID field.)
		return []anypoint.App{{ID: appID}}, nil
	}
	// Otherwise, retrieve all apps.
	return client.GetApps(ctx, orgID, envID, filters...)
}

// monitorSingleApp retrieves monitoring data for a single app.
func monitorSingleApp(ctx context.Context, client *anypoint.Client, orgID, envID string, app anypoint.App, lcWindow, rcWindow string) AppResult {
	var res AppResult
	res.AppID = app.ID
	res.AppType = app.GetType()
	res.LCWindow = lcWindow
	res.RCWindow = rcWindow

	lastCalled, err1 := client.GetLastCalledTime(ctx, orgID, envID, app.ID, lcWindow)
	reqCount, err2 := client.GetRequestCount(ctx, orgID, envID, app.ID, rcWindow)
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

// printSummary prints a condensed summary table for multiple apps.
func printSummary(results []AppResult) {
	fmt.Println("")
	printAppsSummaryTable(results)
}

// printAppsSummaryTable prints a condensed table of app monitoring results
// using tabwriter for alignment.
func printAppsSummaryTable(results []AppResult) {
	// Create a new tabwriter with a minimum width of 0, tab width of 8,
	// padding of 2, and using a tab ('\t') as the padding character.
	w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)

	// Print header row.
	fmt.Fprintln(w, "App ID\tType\tLast Called\tRequest Count")
	fmt.Fprintln(w, "------\t----\t-----------\t-------------")

	// Iterate over the results and print each row.
	for _, r := range results {
		var lastCalled string
		if r.LastCalled.IsZero() {
			lastCalled = "No data"
		} else {
			lastCalled = r.LastCalled.Format(time.RFC1123)
		}
		// Each column is separated by a tab character.
		fmt.Fprintf(w, "%s\t%s\t%s\t%d\n", r.AppID, r.AppType, lastCalled, r.RequestCount)
	}

	// Flush the writer to ensure output is written.
	w.Flush()
}

// printDetailedResult prints detailed monitoring info for a single app.
func printDetailedResult(res AppResult) {
	data := map[string]interface{}{
		"App ID":           res.AppID,
		"Last Called Time": res.LastCalled,
		"Request Count":    res.RequestCount,
		"LC Window":        res.LCWindow,
		"RC Window":        res.RCWindow,
	}
	PrintSimpleResults("Monitoring Results", data)
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
		// Retrieve the context from the command.
		ctx := cmd.Context()

		// Retrieve required flags.
		orgID, _ := cmd.Flags().GetString("org")
		envID, _ := cmd.Flags().GetString("env")
		appID, _ := cmd.Flags().GetString("app")
		lcWindow, _ := cmd.Flags().GetString("last-called-window")
		rcWindow, _ := cmd.Flags().GetString("request-count-window")
		dataFilter, _ := cmd.Flags().GetString("filter")
		appType, _ := cmd.Flags().GetString("app-type")

		// Check that the required flags are provided.
		if orgID == "" || envID == "" {
			fmt.Println("Please provide --org, --env flags")
			return
		}

		// Retrieve the previously connected client from context.
		client, err := anypoint.GetClientFromContext()
		if err != nil {
			fmt.Printf("Error retrieving client: %v\n", err)
			return
		}

		// Display the client info in a colorful way.
		PrintClientInfo(client)

		// Build type filters based on app-type flag.
		var typeFilters []anypoint.AppFilter = []anypoint.AppFilter{anypoint.FilterRunning}
		switch strings.ToLower(appType) {
		case "cloudhub":
			typeFilters = append(typeFilters, anypoint.FilterCloudhub)
		case "rtf":
			typeFilters = append(typeFilters, anypoint.FilterRTF)
			// "all" (or any other value) does not add any type filter.
		}

		// Retrieve apps to monitor.
		apps, err := getAppsToMonitor(ctx, client, orgID, envID, appID, typeFilters...)
		if err != nil {
			fmt.Printf("Error retrieving apps: %v\n", err)
			return
		}

		if len(apps) == 0 {
			fmt.Println("No apps found for the given org and env.")
			return
		}

		// If a single app was specified, run in single-app mode.
		if appID != "" {
			result := monitorSingleApp(ctx, client, orgID, envID, apps[0], lcWindow, rcWindow)
			if result.Err != nil {
				fmt.Printf("Error monitoring app %s: %v\n", appID, result.Err)
				return
			}
			printDetailedResult(result)
			return
		}

		// Monitor all apps concurrently.
		allResults := monitorAppsConcurrently(ctx, client, orgID, envID, lcWindow, rcWindow, apps)
		fmt.Printf("\n* Using last-called window: %s\n", lcWindow)
		fmt.Printf("* Using request count window: %s\n", rcWindow)
		fmt.Printf("* Found %d apps to monitor.\n", len(apps))
		fmt.Printf("* Collected monitoring data for %d apps.\n", len(allResults))

		// Apply filter.
		finalResults := filterAppResults(allResults, dataFilter)
		fmt.Printf("* After applying filter '%s', %d apps remain.\n", dataFilter, len(finalResults))
		if len(finalResults) == 0 {
			fmt.Println("No apps match the filter criteria.")
			return
		}

		// Print a summary if there are multiple apps.
		printSummary(finalResults)

	},
}

func init() {
	// Add the monitor command to the root command.
	rootCmd.AddCommand(monitorCmd)

	// Define flags for organization, environment, and application IDs.
	monitorCmd.Flags().String("org", "", "Organization ID")
	monitorCmd.Flags().String("env", "", "Environment ID")
	monitorCmd.Flags().String("app", "", "Application ID to monitor")

	// Define flags for specifying the time window for queries.
	monitorCmd.Flags().String("last-called-window", "15m", "Time window for last-called query (e.g., 15m, 1h, 24h)")
	monitorCmd.Flags().String("request-count-window", "24h", "Time window for request count query (e.g., 24h, 3d)")

	// Define a flag to filter the results.
	monitorCmd.Flags().String("filter", "all", "Filter results: all (default), nonempty (only apps with monitoring data), or empty (only apps with no data)")
	monitorCmd.Flags().String("app-type", "all", "Filter apps by type: all (default), cloudhub (only CloudHub apps), or rtf (only RTF apps)")

	// Mark the required flags.
	monitorCmd.MarkFlagRequired("org")
	monitorCmd.MarkFlagRequired("env")
}
