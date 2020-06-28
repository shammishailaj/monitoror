package usecase

import (
	"errors"
	"fmt"
	"testing"

	"github.com/monitoror/monitoror/api/config/mocks"
	"github.com/monitoror/monitoror/api/config/models"
	"github.com/monitoror/monitoror/api/config/versions"
	coreConfig "github.com/monitoror/monitoror/config"

	"github.com/stretchr/testify/assert"
	. "github.com/stretchr/testify/mock"
)

func TestConfigUsecase_GetConfigList(t *testing.T) {
	usecase := configUsecase{
		namedConfigs: map[coreConfig.ConfigName]string{
			coreConfig.DefaultConfigName:     "test",
			coreConfig.ConfigName("screen1"): "test2",
		},
	}

	list := usecase.GetConfigList()
	assert.Len(t, list, 2)
	assert.Contains(t, list, models.ConfigMetadata{Name: "default"})
	assert.Contains(t, list, models.ConfigMetadata{Name: "screen1"})
}

func TestUsecase_GetConfig_WithURL_Success(t *testing.T) {
	mockRepo := new(mocks.Repository)
	mockRepo.On("GetConfigFromURL", AnythingOfType("string")).Return(&models.Config{}, nil)

	usecase := initConfigUsecase(mockRepo)

	configBag := usecase.GetConfig(&models.ConfigParams{Config: "http://example.com/config.json"})
	if assert.Len(t, configBag.Errors, 0) {
		mockRepo.AssertNumberOfCalls(t, "GetConfigFromURL", 1)
		mockRepo.AssertExpectations(t)
	}
}

func TestUsecase_GetConfig_WithNamedVariant_Success(t *testing.T) {
	mockRepo := new(mocks.Repository)
	mockRepo.On("GetConfigFromPath", AnythingOfType("string"), AnythingOfType("string")).Return(&models.Config{}, nil)
	mockRepo.On("GetConfigFromURL", AnythingOfType("string")).Return(&models.Config{}, nil)

	usecase := initConfigUsecase(mockRepo)
	usecase.namedConfigs = make(map[coreConfig.ConfigName]string)
	usecase.namedConfigs[coreConfig.DefaultConfigName] = "./config.json"
	usecase.namedConfigs["with-url"] = "http://example.com/config.json"

	configBag := usecase.GetConfig(&models.ConfigParams{Config: string(coreConfig.DefaultConfigName)})
	if assert.Len(t, configBag.Errors, 0) {
		mockRepo.AssertNumberOfCalls(t, "GetConfigFromPath", 1)
	}

	configBag = usecase.GetConfig(&models.ConfigParams{Config: "WITH-URL"})
	if assert.Len(t, configBag.Errors, 0) {
		mockRepo.AssertNumberOfCalls(t, "GetConfigFromPath", 1)
	}
	mockRepo.AssertExpectations(t)
}

func TestUsecase_GetConfig_WithNamedVariant_UnknownConfigName(t *testing.T) {
	mockRepo := new(mocks.Repository)
	usecase := initConfigUsecase(mockRepo)
	usecase.namedConfigs = make(map[coreConfig.ConfigName]string)
	usecase.namedConfigs["test"] = "test"

	configBag := usecase.GetConfig(&models.ConfigParams{})
	if assert.Len(t, configBag.Errors, 1) {
		assert.Equal(t, models.ConfigErrorUnknownNamedConfig, configBag.Errors[0].ID)
		mockRepo.AssertExpectations(t)
	}
}

func TestUsecase_GetConfig_WithError(t *testing.T) {
	for _, testcase := range []struct {
		err       error
		errorID   models.ConfigErrorID
		errorData models.ConfigErrorData
	}{
		{
			err:     errors.New("boom"),
			errorID: models.ConfigErrorUnexpectedError,
		},
		{
			err:       &models.ConfigFileNotFoundError{Err: errors.New("boom"), PathOrURL: "path"},
			errorID:   models.ConfigErrorConfigNotFound,
			errorData: models.ConfigErrorData{Value: "path"},
		},
		{
			err:     &versions.ConfigVersionFormatError{WrongVersion: "18"},
			errorID: models.ConfigErrorUnsupportedVersion,
			errorData: models.ConfigErrorData{
				Value:     "18",
				FieldName: "version",
				Expected:  fmt.Sprintf("%q >= version >= %q", versions.MinimalVersion, versions.CurrentVersion),
			},
		},
		{
			err:       &models.ConfigUnmarshalError{Err: errors.New("boom"), RawConfig: "test json"},
			errorID:   models.ConfigErrorUnableToParseConfig,
			errorData: models.ConfigErrorData{ConfigExtract: "test json"},
		},
		{
			err:       &models.ConfigUnmarshalError{Err: errors.New(`json: unknown field "test"`), RawConfig: "test json"},
			errorID:   models.ConfigErrorUnknownField,
			errorData: models.ConfigErrorData{FieldName: "test", ConfigExtract: "test json", Expected: "version, columns, zoom, tiles, type, label, rowSpan, columnSpan, tiles, url, initialMaxDelay, params, configVariant"},
		},
		{
			err:       &models.ConfigUnmarshalError{Err: errors.New(`json: cannot unmarshal string into Go struct field TileConfig.tiles.test of type int`), RawConfig: "test json"},
			errorID:   models.ConfigErrorFieldTypeMismatch,
			errorData: models.ConfigErrorData{FieldName: "test", ConfigExtract: "test json", Expected: "int"},
		},
		{
			err:       &models.ConfigUnmarshalError{Err: errors.New(`'\s' in string escape code`), RawConfig: "test json"},
			errorID:   models.ConfigErrorInvalidEscapedCharacter,
			errorData: models.ConfigErrorData{ConfigExtract: "test json", ConfigExtractHighlight: `\\s`},
		},
	} {
		mockRepo := new(mocks.Repository)
		mockRepo.On("GetConfigFromPath", AnythingOfType("string"), AnythingOfType("string")).Return(nil, testcase.err)

		usecase := initConfigUsecase(mockRepo)
		usecase.namedConfigs = make(map[coreConfig.ConfigName]string)
		usecase.namedConfigs[coreConfig.DefaultConfigName] = "./config.json"

		configBag := usecase.GetConfig(&models.ConfigParams{Config: string(coreConfig.DefaultConfigName)})
		if assert.Len(t, configBag.Errors, 1) {
			assert.Equal(t, testcase.errorID, configBag.Errors[0].ID)
			assert.Equal(t, testcase.errorData, configBag.Errors[0].Data)
			mockRepo.AssertNumberOfCalls(t, "GetConfigFromPath", 1)
			mockRepo.AssertExpectations(t)
		}
	}
}
