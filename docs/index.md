# Mule Tracker CLI

The mule tracker CLI is a command-line tool written in Go that helps you monitor MuleSoft application activity. It connects to the Anypoint Platform, retrieves app metrics (such as the last time an app was called and the number of requests in a specified time window), and displays the information in a stylish and configurable manner. It also allows you to retrieve a list of applications and filter monitoring results based on whether they have data.

## How to use the CLI

### Help Command

You can get help on the CLI by running:

```bash
./muletracker-cli --help
```

### Autocomplete

You can get completion script for your shell by running:

```bash
./muletracker-cli completion [bash|zsh|fish|powershell]
```

You will need to install the completion script in your local shell.

### Connect to the Anypoint Platform

The connect command allows you to authenticate and establish a connection to the Anypoint Platform through a connected app.

To authenticate and establish a connection, run:

```bash
./muletracker-cli connect --clientId YOUR_CLIENT_ID --clientSecret YOUR_CLIENT_SECRET --controlplane eu
```

If you have previously connected, you can omit the credentials and control plane; they will be read from the configuration file:

```bash
./muletracker-cli connect
```

On success, you will see a confirmation message along with the access token expiration and the InfluxDB ID (retrieved from bootdata).

### Runtime Commands

The `runtime` commands allow you to interact with MuleSoft Runtime Manager and the Monitoring API.

The `runtime` commands has a set of subcommands:

  * `list-apps`: List applications in a specified organization and environment.
  * `monitor`: Monitor applications in a specified organization and environment.

#### List Applications

The `runtime list-apps` command allows you to retrieve a list of applications in a specified organization and environment.

The `runtime list-apps` command has the following flags:

  * org: Organization ID
  * env: Environment ID
  * token: Anypoint Access Token, this is used in case you want to use a different token from the one you have already saved from the connect command.
  * out: Output file for CSV export

To list applications in Runtime Manager run:

```bash
./muletracker-cli runtime list-apps --org YOUR_ORG_ID --env YOUR_ENV_ID
```

The output will be a table with the following columns:

  * App Name
  * App Status
  * Target
  * Mule Version
  * Mule Version EOL

##### Exporting Results

You can export the results to a CSV file using the `--out` flag:

```bash
./muletracker-cli runtime list-apps --org YOUR_ORG_ID --env YOUR_ENV_ID --out results.csv
```

#### Monitor Applications

The `runtime monitor` command allows you to retrieve metrics such as the last time an app was called and the number of requests in a specified time window.

The `runtime monitor` command can be used to monitor a single app or all apps in a specified organization and environment.

The `runtime monitor` command has the following flags:

  * org: Organization ID
  * env: Environment ID
  * app: App ID
  * last-called-window: Time window for last called time (e.g. 15m, 1h)
  * request-count-window: Time window for request count (e.g. 24h)
  * filter: Filter flag (all, nonempty, empty)
  * token: Anypoint Access Token, this is used in case you want to use a different token from the one you have already saved from the connect command.
  * out: Output file for CSV export

##### Monitor a Single App

To monitor a single app, provide the app ID:

```bash
./muletracker-cli runtime monitor --org YOUR_ORG_ID --env YOUR_ENV_ID --app YOUR_APP_ID --last-called-window 15m --request-count-window 24h
```

##### Monitor All Apps

If you omit the `--app` flag, the CLI will retrieve all apps in the specified organization and environment and monitor them concurrently. For example:

```bash
./muletracker-cli runtime monitor --org YOUR_ORG_ID --env YOUR_ENV_ID --last-called-window 15m --request-count-window 24h
```

##### Filtering Results

You can filter the results using the `--filter` flag:

  * all (default): Show all apps
  * nonempty: Show only apps with monitoring data (non-zero request count)
  * empty: Show only apps with no monitoring data

For example, to show only apps with monitoring data:

```bash
./muletracker-cli runtime monitor --org YOUR_ORG_ID --env YOUR_ENV_ID --filter nonempty
```

##### Example Output

When monitoring multiple apps, a summary table is printed:

```markdown
App Monitoring Summary
--------------------------------------------------------------------------------
App Name           Type             Last Called                  Request Count
--------------------------------------------------------------------------------
app-name-2         CloudHub         Wed, 20 Sep 2023 10:05:00    150
app-name-1         RTF              No data                      0
--------------------------------------------------------------------------------
```

When monitoring a single app, a detailed output is shown using a simple results printer.

##### Exporting Results

You can export the results to a CSV file using the `--out` flag:

```bash
./muletracker-cli runtime monitor --org YOUR_ORG_ID --env YOUR_ENV_ID --last-called-window 15m --request-count-window 24h --out results.csv
```

The output will be a CSV file with the following columns:

  * App Name
  * Type
  * Last Called
  * Request Count



#### Api Management Commands

The `apim` commands allow interaction with MuleSoft API Manager:

```bash
muletracker apim [subcommand] [flags]
```

**Subcommands:**

1. **list-apis** - List API Manager instances  
   ```bash
   muletracker apim list-apis [flags]
   ```
   *Flags:*
   - `--org` - Business Group ID (default uses saved context)
   - `--env` - Environment ID (required)
   - `--token` - Admin token for authentication
   - `--type` - Filter by API type (mule3/mule4/flexgateway)
   - `--status` - Filter by API status
   - `--visibility` - Filter by visibility
   - `--output` - Output format (table/json/csv)
   - `--file` - Export results to file

   *Example:*
   ```bash
   muletracker apim list-apis --env production --type mule4 --output csv --file apis.csv
   ```

2. **list-contracts** - List API contracts  
   ```bash
   muletracker apim list-contracts [flags]
   ```
   *Flags:*
   - `--org` - Business Group ID
   - `--env` - Environment ID (required)
   - `--token` - Admin token
   - `--app` - Filter by specific application ID
   - `--output` - Output format
   - `--file` - Export results

   *Example:*
   ```bash
   muletracker apim list-contracts --env staging --app 12345 --output json
   ```

3. **delete-api** - Delete API Manager instance  
   ```bash
   muletracker apim delete-api [flags]
   ```
   *Required Flags:*
   - `--id` - API Manager instance ID to delete
   - `--env` - Environment ID

   *Example:*
   ```bash
   muletracker apim delete-api --env production --id 67890
   ```

**Common Features:**
- All commands support `--token` for authentication override
- Output formats include table (default), JSON, and CSV
- Results can be exported to files using `--file` flag
- Environment ID (`--env`) is always required
- Organization ID (`--org`) defaults to saved context if available

**Notes:**
- When filtering APIs by type, valid values are: mule3, mule4, flexgateway
- Contract listing shows all SLA tiers and client applications
- Delete operations are permanent - use with caution


### Concurrency & Rate Limiting

* Concurrency Limit: Up to 5 monitoring requests are executed concurrently.
* Rate Limit: The CLI enforces a maximum of 10 monitoring requests per second.

These limits help prevent overwhelming the API endpoints.

## Command Reference

### Authentication
`muletracker connect`  
Authenticate with Anypoint Platform using client credentials

### Environment Management
`muletracker environments`  
List and manage available environments

### Exchange Operations
```
muletracker exchange create-client-app
muletracker exchange list-client-apps
muletracker exchange delete-client-app
```
Manage Exchange client applications

### Runtime Management
```
muletracker runtime list       # List deployed applications
muletracker runtime monitor    # Monitor application metrics
```

### API Management
```
muletracker apim list-apis       # List published APIs
muletracker apim list-contracts  # List API contracts
muletracker apim delete-api      # Remove an API
```
