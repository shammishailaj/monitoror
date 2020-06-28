package usecase

import (
	"errors"
	"testing"
	"time"

	"github.com/monitoror/monitoror/monitorables/jenkins/api"

	coreModels "github.com/monitoror/monitoror/models"
	"github.com/monitoror/monitoror/monitorables/jenkins/api/mocks"
	"github.com/monitoror/monitoror/monitorables/jenkins/api/models"
	"github.com/monitoror/monitoror/pkg/git"

	. "github.com/AlekSi/pointer"
	"github.com/stretchr/testify/assert"
	. "github.com/stretchr/testify/mock"
)

var job, branch = "test", "master"

func TestBuild_Error(t *testing.T) {
	mockRepository := new(mocks.Repository)
	mockRepository.On("GetJob", AnythingOfType("string"), AnythingOfType("string")).
		Return(nil, errors.New("boom"))

	tu := NewJenkinsUsecase(mockRepository)

	tile, err := tu.Build(&models.BuildParams{Job: job, Branch: branch})
	if assert.Error(t, err) {
		assert.Nil(t, tile)
		assert.IsType(t, &coreModels.MonitororError{}, err)
		assert.Equal(t, "unable to find job", err.Error())
		mockRepository.AssertNumberOfCalls(t, "GetJob", 1)
		mockRepository.AssertExpectations(t)
	}
}

func TestBuild_DisabledBuild(t *testing.T) {
	repositoryJob := &models.Job{
		Buildable: false,
	}

	mockRepository := new(mocks.Repository)
	mockRepository.On("GetJob", AnythingOfType("string"), AnythingOfType("string")).
		Return(repositoryJob, nil)

	tu := NewJenkinsUsecase(mockRepository)

	tile, err := tu.Build(&models.BuildParams{Job: job})
	if assert.NoError(t, err) {
		assert.Equal(t, job, tile.Label)
		assert.Equal(t, coreModels.DisabledStatus, tile.Status)
		mockRepository.AssertNumberOfCalls(t, "GetJob", 1)
		mockRepository.AssertExpectations(t)
	}
}

func TestBuild_Error_NoBuild(t *testing.T) {
	repositoryJob := &models.Job{
		Buildable: true,
	}

	mockRepository := new(mocks.Repository)
	mockRepository.On("GetJob", AnythingOfType("string"), AnythingOfType("string")).
		Return(repositoryJob, nil)
	mockRepository.On("GetLastBuildStatus", Anything).
		Return(nil, errors.New("boom"))

	tu := NewJenkinsUsecase(mockRepository)

	tile, err := tu.Build(&models.BuildParams{Job: job, Branch: branch})
	if assert.Error(t, err) {
		assert.Nil(t, tile)
		assert.IsType(t, &coreModels.MonitororError{}, err)
		assert.Equal(t, "no build found", err.Error())
		assert.Equal(t, coreModels.UnknownStatus, err.(*coreModels.MonitororError).ErrorStatus)
		mockRepository.AssertNumberOfCalls(t, "GetJob", 1)
		mockRepository.AssertNumberOfCalls(t, "GetLastBuildStatus", 1)
		mockRepository.AssertExpectations(t)
	}
}

func CheckBuild(t *testing.T, result string) {
	repositoryJob := &models.Job{
		Buildable: true,
	}
	repositoryBuild := buildResponse(result, time.Date(2000, 01, 01, 10, 00, 00, 00, time.UTC), time.Minute)

	mockRepository := new(mocks.Repository)
	mockRepository.On("GetJob", AnythingOfType("string"), AnythingOfType("string")).
		Return(repositoryJob, nil)
	mockRepository.On("GetLastBuildStatus", Anything).
		Return(repositoryBuild, nil)

	tu := NewJenkinsUsecase(mockRepository)
	tUsecase, ok := tu.(*jenkinsUsecase)
	if assert.True(t, ok, "enable to case tu into travisCIUsecase") {
		expected := coreModels.NewTile(api.JenkinsBuildTileType).WithBuild()
		expected.Label = job
		expected.Build.ID = ToString("1")
		expected.Build.Branch = ToString(git.HumanizeBranch(branch))

		expected.Status = parseResult(repositoryBuild.Result)
		expected.Build.PreviousStatus = coreModels.SuccessStatus
		expected.Build.StartedAt = ToTime(repositoryBuild.StartedAt)
		expected.Build.FinishedAt = ToTime(repositoryBuild.StartedAt.Add(repositoryBuild.Duration))

		if result == "FAILURE" {
			expected.Build.Author = &coreModels.Author{
				Name:      repositoryBuild.Author.Name,
				AvatarURL: repositoryBuild.Author.AvatarURL,
			}
		}

		// Add cache for previousStatus
		params := &models.BuildParams{Job: job, Branch: branch}
		tUsecase.buildsCache.Add(params, "0", coreModels.SuccessStatus, time.Second*120)
		tile, err := tu.Build(params)
		if assert.NoError(t, err) {
			assert.Equal(t, expected, tile)
			mockRepository.AssertNumberOfCalls(t, "GetJob", 1)
			mockRepository.AssertNumberOfCalls(t, "GetLastBuildStatus", 1)
			mockRepository.AssertExpectations(t)
		}
	}
}

func TestBuild_Success(t *testing.T) {
	CheckBuild(t, "SUCCESS")
}

func TestBuild_Unstable(t *testing.T) {
	CheckBuild(t, "UNSTABLE")
}

func TestBuild_Failure(t *testing.T) {
	CheckBuild(t, "FAILURE")
}

func TestBuild_Aborted(t *testing.T) {
	CheckBuild(t, "ABORTED")
}

func TestBuild_Queued(t *testing.T) {
	repositoryJob := &models.Job{
		Buildable: true,
		InQueue:   true,
		QueuedAt:  ToTime(time.Date(2000, 01, 01, 10, 00, 00, 00, time.UTC)),
	}

	mockRepository := new(mocks.Repository)
	mockRepository.On("GetJob", AnythingOfType("string"), AnythingOfType("string")).
		Return(repositoryJob, nil)

	tu := NewJenkinsUsecase(mockRepository)
	tUsecase, ok := tu.(*jenkinsUsecase)
	if assert.True(t, ok, "enable to case tu into travisCIUsecase") {
		expected := coreModels.NewTile(api.JenkinsBuildTileType).WithBuild()
		expected.Label = job
		expected.Build.Branch = ToString(git.HumanizeBranch(branch))

		expected.Status = coreModels.QueuedStatus
		expected.Build.PreviousStatus = coreModels.SuccessStatus
		expected.Build.StartedAt = repositoryJob.QueuedAt

		// Add cache for previousStatus
		params := &models.BuildParams{Job: job, Branch: branch}
		tUsecase.buildsCache.Add(params, "0", coreModels.SuccessStatus, time.Second*120)
		tile, err := tu.Build(params)
		if assert.NoError(t, err) {
			assert.Equal(t, expected, tile)
			mockRepository.AssertNumberOfCalls(t, "GetJob", 1)
			mockRepository.AssertExpectations(t)
		}
	}
}

func TestBuild_Running(t *testing.T) {
	repositoryJob := &models.Job{
		Buildable: true,
	}
	repositoryBuild := buildResponse("null", time.Now(), 0)
	repositoryBuild.Building = true

	mockRepository := new(mocks.Repository)
	mockRepository.On("GetJob", AnythingOfType("string"), AnythingOfType("string")).
		Return(repositoryJob, nil)
	mockRepository.On("GetLastBuildStatus", Anything).
		Return(repositoryBuild, nil)

	ju := NewJenkinsUsecase(mockRepository)
	jUsecase, ok := ju.(*jenkinsUsecase)
	if assert.True(t, ok, "enable to case ju into jenkinsUsecase") {
		// Without cached build
		expected := coreModels.NewTile(api.JenkinsBuildTileType).WithBuild()
		expected.Label = job
		expected.Build.ID = ToString("1")
		expected.Build.Branch = ToString(git.HumanizeBranch(branch))

		expected.Status = coreModels.RunningStatus
		expected.Build.PreviousStatus = coreModels.UnknownStatus
		expected.Build.StartedAt = ToTime(repositoryBuild.StartedAt)
		expected.Build.Duration = ToInt64(int64(0))
		expected.Build.EstimatedDuration = ToInt64(int64(0))

		params := &models.BuildParams{Job: job, Branch: branch}
		tile, err := ju.Build(params)
		if assert.NoError(t, err) {
			assert.Equal(t, expected, tile)
		}

		// With cached build
		jUsecase.buildsCache.Add(params, "0", coreModels.SuccessStatus, time.Second*120)

		expected.Build.PreviousStatus = coreModels.SuccessStatus
		expected.Build.EstimatedDuration = ToInt64(int64(120))

		tile, err = ju.Build(params)
		if assert.NoError(t, err) {
			assert.Equal(t, expected, tile)
		}

		mockRepository.AssertNumberOfCalls(t, "GetJob", 2)
		mockRepository.AssertNumberOfCalls(t, "GetLastBuildStatus", 2)
		mockRepository.AssertExpectations(t)
	}
}

func TestBuildGenerator_Success(t *testing.T) {
	repositoryJob := &models.Job{
		ID:        job,
		Buildable: false,
		InQueue:   false,
		Branches:  []string{branch, "develop", "feat%2Ftest-deploy"},
	}

	mockRepository := new(mocks.Repository)
	mockRepository.On("GetJob", AnythingOfType("string"), AnythingOfType("string")).
		Return(repositoryJob, nil)

	tu := NewJenkinsUsecase(mockRepository)

	tiles, err := tu.BuildGenerator(&models.BuildGeneratorParams{Job: job})
	if assert.NoError(t, err) {
		assert.Len(t, tiles, 3)
		params, ok := tiles[0].Params.(*models.BuildParams)
		if assert.True(t, ok) {
			assert.Equal(t, job, params.Job)
			assert.Equal(t, "master", params.Branch)
		}
		params, ok = tiles[1].Params.(*models.BuildParams)
		if assert.True(t, ok) {
			assert.Equal(t, job, params.Job)
			assert.Equal(t, "develop", params.Branch)
		}
		params, ok = tiles[2].Params.(*models.BuildParams)
		if assert.True(t, ok) {
			assert.Equal(t, job, params.Job)
			assert.Equal(t, "feat%2Ftest-deploy", params.Branch)
		}
	}

	tiles, err = tu.BuildGenerator(&models.BuildGeneratorParams{Job: job, Match: "feat/*"})
	if assert.NoError(t, err) {
		assert.Len(t, tiles, 1)
		params, ok := tiles[0].Params.(*models.BuildParams)
		if assert.True(t, ok) {
			assert.Equal(t, job, params.Job)
			assert.Equal(t, "feat%2Ftest-deploy", params.Branch)
		}
	}

	mockRepository.AssertNumberOfCalls(t, "GetJob", 2)
	mockRepository.AssertExpectations(t)
}

func TestBuildGenerator_Error(t *testing.T) {
	mockRepository := new(mocks.Repository)
	mockRepository.On("GetJob", AnythingOfType("string"), AnythingOfType("string")).
		Return(nil, errors.New("boom"))

	tu := NewJenkinsUsecase(mockRepository)

	_, err := tu.BuildGenerator(&models.BuildGeneratorParams{Job: "test"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unable to find job")

	mockRepository.AssertNumberOfCalls(t, "GetJob", 1)
	mockRepository.AssertExpectations(t)
}

func TestBuildGenerator_ErrorWithRegex(t *testing.T) {
	mockRepository := new(mocks.Repository)
	mockRepository.On("GetJob", AnythingOfType("string"), AnythingOfType("string")).
		Return(nil, nil)

	tu := NewJenkinsUsecase(mockRepository)

	_, err := tu.BuildGenerator(&models.BuildGeneratorParams{Job: "test", Match: "("})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error parsing regexp")

	_, err = tu.BuildGenerator(&models.BuildGeneratorParams{Job: "test", Unmatch: "("})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error parsing regexp")

	mockRepository.AssertNumberOfCalls(t, "GetJob", 2)
	mockRepository.AssertExpectations(t)
}

func TestParseResult(t *testing.T) {
	assert.Equal(t, coreModels.SuccessStatus, parseResult("SUCCESS"))
	assert.Equal(t, coreModels.WarningStatus, parseResult("UNSTABLE"))
	assert.Equal(t, coreModels.FailedStatus, parseResult("FAILURE"))
	assert.Equal(t, coreModels.CanceledStatus, parseResult("ABORTED"))
	assert.Equal(t, coreModels.UnknownStatus, parseResult(""))
}

func buildResponse(result string, startedAt time.Time, duration time.Duration) *models.Build {
	repositoryBuild := &models.Build{
		Number:    "1",
		FullName:  "Test-Build",
		Result:    result,
		StartedAt: startedAt,
		Duration:  duration,
		Author: &coreModels.Author{
			Name:      "me",
			AvatarURL: "http://avatar.com",
		},
	}
	return repositoryBuild
}
