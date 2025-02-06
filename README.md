# MuleTracker cli

MuleTracker CLI is a command-line tool written in Go that helps you monitor MuleSoft application activity. It connects to the Anypoint Platform, retrieves app metrics (such as the last time an app was called and the number of requests in a specified time window), and displays the information in a stylish and configurable manner. It also allows you to retrieve a list of applications and filter monitoring results based on whether they have data.

## Features
* Connect to Anypoint Platform:
Authenticate with your MuleSoft connected app credentials and persist configuration (client ID, client secret, control plane, access token, etc.) using Viper.

* Monitor Application Metrics:
Retrieve metrics like the last-called time (using the 75th percentile of average response time) and request count over configurable time windows.

* Monitor All Apps:
If no specific app is provided, the CLI retrieves a list of all applications in a given organization and environment and monitors them concurrently with rate limiting and concurrency controls.

* Filtering Results:
Use a filter flag to display all apps, only those with monitoring data, or only those with no data.

* Stylish Output:
Results are printed in a clear, aligned, and colorful format using a simple, AWS CLI–style table output.

## Requirements
* Go (version 1.16 or later recommended)
* Internet access to query the Anypoint Platform endpoints

## Installation
1. Clone the repository:

```bash
git clone https://github.com/mulesoft-anypoint/muletracker-cli.git
cd muletracker-cli
```

2. Install dependencies:

The project uses several Go modules including:

- Cobra for CLI scaffolding.
- Viper for configuration management.
- Fatih/color for colorful output.

Run the following command to download dependencies:

```bash
go mod tidy
```

3. Build the CLI:

```bash
go build -o muletracker-cli
```

## Configuration

Upon first run, the connect command will prompt you to provide your connected app credentials (client ID, client secret) and control plane (e.g. eu, us, or gov). These details, along with the access token and its expiration, are persisted in a configuration file (by default at `$HOME/.muletracker.yaml`).

> **Security Notice**:
> For production use, consider using a more secure method to store sensitive credentials.

## Usage
### Connect to the Anypoint Platform
To authenticate and establish a connection, run:

```bash
./muletracker-cli connect --clientId YOUR_CLIENT_ID --clientSecret YOUR_CLIENT_SECRET --controlplane eu
```

If you have previously connected, you can omit the credentials and control plane; they will be read from the configuration file:

```bash
./muletracker-cli connect
```

On success, you will see a confirmation message along with the access token expiration and the InfluxDB ID (retrieved from bootdata).

### Monitor Applications

#### Monitor a Single App

To monitor a single app, provide the app ID:

```bash
./muletracker-cli monitor --org YOUR_ORG_ID --env YOUR_ENV_ID --app YOUR_APP_ID --last-called-window 15m --request-count-window 24h
```

#### Monitor All Apps
If you omit the --app flag, the CLI will retrieve all apps in the specified organization and environment and monitor them concurrently. For example:

```bash
./muletracker-cli monitor --org YOUR_ORG_ID --env YOUR_ENV_ID --last-called-window 15m --request-count-window 24h
```

#### Filtering Results
You can filter the results using the --filter flag:

  * all (default): Show all apps
  * nonempty: Show only apps with monitoring data (non-zero request count)
  * empty: Show only apps with no monitoring data

For example, to show only apps with monitoring data:

```bash
./muletracker-cli monitor --org YOUR_ORG_ID --env YOUR_ENV_ID --filter nonempty
```

#### Example Output
When monitoring multiple apps, a summary table is printed:

```markdown
App Monitoring Summary
--------------------------------------------------------------------------------
App ID                          Last Called                  Request Count
--------------------------------------------------------------------------------
app-name-2                      Wed, 20 Sep 2023 10:05:00    150
app-name-1                      No data                      0
--------------------------------------------------------------------------------
```

When monitoring a single app, a detailed output is shown using a simple results printer.

## Concurrency & Rate Limiting
* Concurrency Limit: Up to 5 monitoring requests are executed concurrently.
* Rate Limit: The CLI enforces a maximum of 10 monitoring requests per second.

These limits help prevent overwhelming the API endpoints.


## Contributing

Contributions are welcome! Please open an issue or submit a pull request if you have improvements or bug fixes.

## License

MIT License