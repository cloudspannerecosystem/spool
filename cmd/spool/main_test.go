package main

import (
	"os"
	"testing"
)

func TestLoadEnvVarsIfNeeded(t *testing.T) {
	tests := map[string]struct {
		setup   func()
		envVars map[string]string
		assert  func(t *testing.T)
		fail    bool
	}{
		"from env vars": {
			setup: func() {
				projectID = ptr("")
				instanceID = ptr("")
				databaseID = ptr("")
			},
			envVars: map[string]string{
				envGoogleCloudProjectID: "projectID-from-google-cloud-env",
				envInstanceID:           "instanceID-from-env",
				envDatabaseID:           "databaseID-from-env",
			},
			assert: func(t *testing.T) {
				if expected, got := "projectID-from-google-cloud-env", *projectID; expected != got {
					t.Errorf("expected projectID %s but got %s", expected, got)
				}
				if expected, got := "instanceID-from-env", *instanceID; expected != got {
					t.Errorf("expected instanceID %s but got %s", expected, got)
				}
				if expected, got := "databaseID-from-env", *databaseID; expected != got {
					t.Errorf("expected databaseID %s but got %s", expected, got)
				}
			},
		},
		"from env vars (SPANNER_PROJECT_ID overrides GOOGLE_CLOUD_PROJECT)": {
			setup: func() {
				projectID = ptr("")
				instanceID = ptr("")
				databaseID = ptr("")
			},
			envVars: map[string]string{
				envProjectID:            "projectID-from-env",
				envGoogleCloudProjectID: "projectID-from-google-cloud-env",
				envInstanceID:           "instanceID-from-env",
				envDatabaseID:           "databaseID-from-env",
			},
			assert: func(t *testing.T) {
				if expected, got := "projectID-from-env", *projectID; expected != got {
					t.Errorf("expected projectID %s but got %s", expected, got)
				}
				if expected, got := "instanceID-from-env", *instanceID; expected != got {
					t.Errorf("expected instanceID %s but got %s", expected, got)
				}
				if expected, got := "databaseID-from-env", *databaseID; expected != got {
					t.Errorf("expected databaseID %s but got %s", expected, got)
				}
			},
		},
		"from flags": {
			setup: func() {
				projectID = ptr("projectID")
				instanceID = ptr("instanceID")
				databaseID = ptr("databaseID")
			},
			envVars: map[string]string{
				envProjectID:            "projectID-from-env",
				envGoogleCloudProjectID: "projectID-from-google-cloud-env",
				envInstanceID:           "instanceID-from-env",
				envDatabaseID:           "databaseID-from-env",
			},
			assert: func(t *testing.T) {
				if expected, got := "projectID", *projectID; expected != got {
					t.Errorf("expected projectID %s but got %s", expected, got)
				}
				if expected, got := "instanceID", *instanceID; expected != got {
					t.Errorf("expected instanceID %s but got %s", expected, got)
				}
				if expected, got := "databaseID", *databaseID; expected != got {
					t.Errorf("expected databaseID %s but got %s", expected, got)
				}
			},
		},
		"projectID is required": {
			setup: func() {
				projectID = ptr("")
				instanceID = ptr("instanceID")
				databaseID = ptr("databaseID")
			},
			fail: true,
		},
		"instanceID is required": {
			setup: func() {
				projectID = ptr("projectID")
				instanceID = ptr("")
				databaseID = ptr("databaseID")
			},
			fail: true,
		},
		"databaseID is required": {
			setup: func() {
				projectID = ptr("projectID")
				instanceID = ptr("instanceID")
				databaseID = ptr("")
			},
			fail: true,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			test.setup()
			os.Clearenv()
			for k, v := range test.envVars {
				t.Setenv(k, v)
			}
			err := loadEnvVarsIfNeeded()
			if test.fail {
				if err == nil {
					t.Fatal("expected error but no error")
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %s", err)
				}
				if projectID == nil || instanceID == nil || databaseID == nil {
					t.Fatalf("all flags are required: projectID=%v, instanceID=%v, databaseID=%v", projectID, instanceID, databaseID)
				}
				test.assert(t)
			}
		})
	}
}

func ptr(s string) *string {
	return &s
}
