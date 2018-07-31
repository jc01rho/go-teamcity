package teamcity_test

import (
	"testing"

	teamcity "github.com/cvbarros/go-teamcity-sdk/pkg/teamcity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildType_CreateBasicProject(t *testing.T) {
	client := setup()
	newProject := getTestProjectData(testBuildTypeProjectId)
	_, err := client.Projects.Create(newProject)

	if err != nil {
		t.Fatalf("Failed to create project for buildType: %s", err)
	}
	newBuildType := getTestBuildTypeData("PullRequest", "Description", testBuildTypeProjectId)

	actual, err := client.BuildTypes.Create(testBuildTypeProjectId, newBuildType)

	if err != nil {
		t.Fatalf("Failed to CreateBuildType: %s", err)
	}

	if actual == nil {
		t.Fatalf("CreateBuildType did not return a valid instance")
	}

	cleanUpProject(t, client, testBuildTypeProjectId)

	assert.NotEmpty(t, actual.ID)
	assert.Equal(t, newBuildType.ProjectID, actual.ProjectID)
	assert.Equal(t, newBuildType.Name, actual.Name)
}

func TestBuildType_AttachVcsRoot(t *testing.T) {
	client := setup()
	newProject := getTestProjectData(testBuildTypeProjectId)

	if _, err := client.Projects.Create(newProject); err != nil {
		t.Fatalf("Failed to create project for buildType: %s", err)
	}

	newBuildType := getTestBuildTypeData("PullRequest", "Description", testBuildTypeProjectId)

	createdBuildType, err := client.BuildTypes.Create(testBuildTypeProjectId, newBuildType)
	if err != nil {
		t.Fatalf("Failed to CreateBuildType: %s", err)
	}

	newVcsRoot := getTestVcsRootData(testBuildTypeProjectId)

	vcsRootCreated, err := client.VcsRoots.Create(testBuildTypeProjectId, newVcsRoot)

	if err != nil {
		t.Fatalf("Failed to create vcs root: %s", err)
	}

	err = client.BuildTypes.AttachVcsRoot(createdBuildType.ID, vcsRootCreated)
	if err != nil {
		t.Fatalf("Failed to attach vcsRoot '%s' to buildType '%s': %s", createdBuildType.ID, vcsRootCreated.ID, err)
	}

	actual, err := client.BuildTypes.GetByID(createdBuildType.ID)
	if err != nil {
		t.Fatalf("Failed to get buildType '%s' for asserting: %s", createdBuildType.ID, err)
	}

	assert.NotEqualf(t, actual.VcsRootEntries.Count, 0, "Expected VcsRootEntries to contain at least one element")
	vcsEntries := idMapVcsRootEntries(actual.VcsRootEntries)
	assert.Containsf(t, vcsEntries, vcsRootCreated.ID, "Expected VcsRootEntries to contain the VcsRoot with id = %s, but it did not", vcsRootCreated.ID)

	cleanUpProject(t, client, testBuildTypeProjectId)
}

func TestBuildType_AddStepPowerShell(t *testing.T) {
	client := setup()
	step, _ := teamcity.NewStepPowershellScriptFile("step1", "build.ps1", "")
	_, created := createTestBuildStep(t, client, step, testBuildTypeProjectId)

	cleanUpProject(t, client, testBuildTypeProjectId)

	assert.NotNil(t, created)
}

func TestBuildType_AddStepCommandLineExecutable(t *testing.T) {
	assert := assert.New(t)
	client := setup()
	step, _ := teamcity.NewStepCommandLineExecutable("step_exe", "./script.sh", "hello")
	_, actual := createTestBuildStep(t, client, step, testBuildTypeProjectId)

	cleanUpProject(t, client, testBuildTypeProjectId)

	require.NotNil(t, actual)
	assert.Equal(teamcity.StepTypeCommandLine, actual.Type())
}

func TestBuildType_AddStepCommandLineScript(t *testing.T) {
	assert := assert.New(t)
	client := setup()
	script := `echo "Hello World
	echo "World, Hello!
	export HELLO_WORLD=1
	`
	step, _ := teamcity.NewStepCommandLineScript("step_exe", script)
	_, actual := createTestBuildStep(t, client, step, testBuildTypeProjectId)

	cleanUpProject(t, client, testBuildTypeProjectId)

	require.NotNil(t, actual)
	assert.Equal(teamcity.StepTypeCommandLine, actual.Type())
}

func TestBuildType_GetSteps(t *testing.T) {
	client := setup()
	step1, _ := teamcity.NewStepCommandLineExecutable("step1", "./script.sh", "hello")
	step2, _ := teamcity.NewStepCommandLineExecutable("step2", "./script.sh", "hello")

	buildType := createTestBuildType(t, client, testBuildTypeProjectId)

	created1, _ := client.BuildTypes.AddStep(buildType.ID, step1)
	created2, _ := client.BuildTypes.AddStep(buildType.ID, step2)

	steps, _ := client.BuildTypes.GetSteps(buildType.ID)

	cleanUpProject(t, client, testBuildTypeProjectId)

	assert.Contains(t, steps, created1)
	assert.Contains(t, steps, created2)
}

func TestBuildType_DeleteStep(t *testing.T) {
	client := setup()
	step, _ := teamcity.NewStepCommandLineExecutable("step_exe", "./script.sh", "hello")
	buildType, created := createTestBuildStep(t, client, step, testBuildTypeProjectId)

	client.BuildTypes.DeleteStep(buildType.ID, created.ID())

	steps, _ := client.BuildTypes.GetSteps(buildType.ID)

	cleanUpProject(t, client, testBuildTypeProjectId)

	assert.Empty(t, steps)
}

func TestBuildType_UpdateSettings(t *testing.T) {
	client := setup()
	assert := assert.New(t)

	buildType := createTestBuildType(t, client, testBuildTypeProjectId)
	builder := teamcity.BuildTypeSettingsBuilder

	settings := builder.ConfigurationType("composite").
		PersonalBuildTrigger(false).
		ArtifactRules("**/*.zip").
		Build()

	err := client.BuildTypes.UpdateSettings(buildType.ID, settings)
	assert.Nil(err)

	buildType, _ = client.BuildTypes.GetByID(buildType.ID) //refresh
	cleanUpProject(t, client, testBuildTypeProjectId)

	actual := buildType.Settings.Map()

	assert.Equal("COMPOSITE", actual["buildConfigurationType"])
	assert.Equal("false", actual["allowPersonalBuildTriggering"])
	assert.Equal("**/*.zip", actual["artifactRules"])
}

func idMapVcsRootEntries(v *teamcity.VcsRootEntries) map[string]string {
	out := make(map[string]string)
	for _, item := range v.Items {
		out[item.VcsRoot.ID] = item.ID
	}

	return out
}

func createTestBuildType(t *testing.T, client *teamcity.Client, buildTypeProjectId string) *teamcity.BuildType {
	return createTestBuildTypeWithName(t, client, buildTypeProjectId, "PullRequest", true)
}

func createTestBuildTypeWithName(t *testing.T, client *teamcity.Client, buildTypeProjectId string, name string, createProject bool) *teamcity.BuildType {
	if createProject {
		newProject := getTestProjectData(buildTypeProjectId)

		if _, err := client.Projects.Create(newProject); err != nil {
			t.Fatalf("Failed to create project for buildType: %s", err)
		}
	}

	newBuildType := getTestBuildTypeData(name, "Inspection", buildTypeProjectId)

	createdBuildType, err := client.BuildTypes.Create(buildTypeProjectId, newBuildType)
	if err != nil {
		t.Fatalf("Failed to CreateBuildType: %s", err)
	}

	detailed, _ := client.BuildTypes.GetByID(createdBuildType.ID)
	return detailed
}

func createTestBuildStep(t *testing.T, client *teamcity.Client, step teamcity.Step, buildTypeProjectId string) (*teamcity.BuildType, teamcity.Step) {
	createdBuildType := createTestBuildType(t, client, buildTypeProjectId)

	created, err := client.BuildTypes.AddStep(createdBuildType.ID, step)
	if err != nil {
		t.Fatalf("Failed to add step to buildType '%s'", createdBuildType.ID)
	}

	updated, _ := client.BuildTypes.GetByID(createdBuildType.ID)
	return updated, created
}

func getTestBuildTypeData(name string, description string, projectId string) *teamcity.BuildType {

	return &teamcity.BuildType{
		Name:        name,
		Description: description,
		ProjectID:   projectId,
	}
}

func cleanUpBuildType(t *testing.T, c *teamcity.Client, id string) {
	if err := c.BuildTypes.Delete(id); err != nil {
		t.Errorf("Unable to delete build type with id = '%s', err: %s", id, err)
		return
	}

	deleted, err := c.BuildTypes.GetByID(id)

	if deleted != nil {
		t.Errorf("Build type not deleted during cleanup.")
		return
	}

	if err == nil {
		t.Errorf("Expected 404 Not Found error when getting Deleted Build Type, but no error returned.")
	}
}
