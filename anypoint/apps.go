package anypoint

// App represents an application as returned by the ARMUI endpoint.
type App struct {
	ID     string `json:"id"`
	Target struct {
		Type    string `json:"type"`
		Subtype string `json:"subtype,omitempty"`
		ID      string `json:"id,omitempty"`
	} `json:"target"`
	Artifact struct {
		LastUpdateTime int64  `json:"lastUpdateTime"`
		CreateTime     *int64 `json:"createTime"`
		Name           string `json:"name"`
		FileName       string `json:"fileName"`
	} `json:"artifact"`
	MuleVersion struct {
		Version          string `json:"version"`
		UpdateId         string `json:"updateId"`
		LatestUpdateId   string `json:"latestUpdateId"`
		EndOfSupportDate int64  `json:"endOfSupportDate"`
	} `json:"muleVersion"`
	IsDeploymentWaiting bool   `json:"isDeploymentWaiting"`
	LastReportedStatus  string `json:"lastReportedStatus"`
	Application         struct {
		Status string `json:"status"`
	} `json:"application,omitempty"`
	Details struct {
		Domain string `json:"domain,omitempty"`
	} `json:"details"`
}

func (a App) GetType() string {
	if a.Target.Type == "MC" {
		return a.Target.Subtype
	}
	return a.Target.Type
}

// AppsResponse models the response from the applications endpoint.
type AppsResponse struct {
	Data  []App `json:"data"`
	Total int   `json:"total"`
}

// AppFilter defines a function type for filtering apps.
type AppFilter func(app App) bool

// FilterApps returns a slice of apps that match all provided filters.
func FilterApps(apps []App, filters ...AppFilter) []App {
	var filtered []App
	for _, app := range apps {
		match := true
		for _, filter := range filters {
			if !filter(app) {
				match = false
				break
			}
		}
		if match {
			filtered = append(filtered, app)
		}
	}
	return filtered
}

// FilterCloudhub returns true if an app is deployed to CloudHub.
func FilterCH1(app App) bool {
	return app.Target.Type == "CLOUDHUB"
}

// FilterRTF returns true if an app is deployed to RTF (runtime fabrics).
func FilterRTF(app App) bool {
	return app.Target.Type == "MC" && app.Target.Subtype == "runtime-fabric"
}

func FilterCH1OrRTF(app App) bool {
	return FilterCH1(app) || FilterRTF(app)
}

// FilterRunning returns true if an app is running.
func FilterRunning(app App) bool {
	// Check for CloudHub apps.
	if FilterCH1(app) {
		return app.LastReportedStatus == "STARTED"
	}
	// Check for RTF apps.
	if FilterRTF(app) {
		return app.Application.Status == "RUNNING"
	}
	// For other types, do not filter them out.
	return true
}

func FilterByName(name string) AppFilter {
	return func(app App) bool {
		return app.Artifact.Name == name
	}
}
