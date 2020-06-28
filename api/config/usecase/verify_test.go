package usecase

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/monitoror/monitoror/api/config/models"
	"github.com/monitoror/monitoror/api/config/versions"
	"github.com/monitoror/monitoror/internal/pkg/validator"
	coreModels "github.com/monitoror/monitoror/models"
	jenkinsApi "github.com/monitoror/monitoror/monitorables/jenkins/api"
	jenkinsModels "github.com/monitoror/monitoror/monitorables/jenkins/api/models"

	"github.com/stretchr/testify/assert"
)

func initConfig(t *testing.T, rawConfig string) (tiles *models.TileConfig, conf *models.ConfigBag) {
	tiles = &models.TileConfig{}

	err := json.Unmarshal([]byte(rawConfig), &tiles)
	assert.NoError(t, err)

	conf = &models.ConfigBag{Config: &models.Config{Version: versions.CurrentVersion.ToConfigVersion()}}

	return
}

func TestUsecase_Verify_Success(t *testing.T) {
	rawConfig := fmt.Sprintf(`
{
	"version" : %q,
  "columns": 4,
	"zoom": 2.5,
  "tiles": [
		{ "type": "EMPTY" }
  ]
}
`, versions.CurrentVersion)

	conf, err := readConfig(rawConfig)
	if assert.NoError(t, err) {
		usecase := initConfigUsecase(nil)
		usecase.Verify(conf)

		assert.Len(t, conf.Errors, 0)
	}
}

func TestUsecase_Verify_SuccessWithOptionalParameters(t *testing.T) {
	rawConfig := fmt.Sprintf(`
{
	"version" : %q,
  "columns": 4,
  "zoom": 2.5,
  "tiles": [
		{ "type": "EMPTY" }
  ]
}
`, versions.CurrentVersion)

	conf, err := readConfig(rawConfig)

	if assert.NoError(t, err) {
		usecase := initConfigUsecase(nil)
		usecase.Verify(conf)

		assert.Len(t, conf.Errors, 0)
	}
}

func TestUsecase_Verify_Failed(t *testing.T) {
	for _, testcase := range []struct {
		rawConfig string
		errorID   models.ConfigErrorID
		errorData models.ConfigErrorData
	}{
		{
			rawConfig: `{}`,
			errorID:   models.ConfigErrorMissingRequiredField,
			errorData: models.ConfigErrorData{FieldName: "version"},
		},
		{
			rawConfig: `{"version": "0.0"}`,
			errorID:   models.ConfigErrorUnsupportedVersion,
			errorData: models.ConfigErrorData{
				FieldName: "version",
				Value:     `"0.0"`,
				Expected:  fmt.Sprintf(`%q <= version <= %q`, versions.MinimalVersion, versions.CurrentVersion),
			},
		},
		{
			rawConfig: `{"version": "999.999"}`, // Don't let me go that far ^^'
			errorID:   models.ConfigErrorUnsupportedVersion,
			errorData: models.ConfigErrorData{
				FieldName: "version",
				Value:     `"999.999"`,
				Expected:  fmt.Sprintf(`%q <= version <= %q`, versions.MinimalVersion, versions.CurrentVersion),
			},
		},
		{
			rawConfig: fmt.Sprintf(`{"version": %q, "tiles": [{ "type": "EMPTY" }]}`, versions.CurrentVersion),
			errorID:   models.ConfigErrorMissingRequiredField,
			errorData: models.ConfigErrorData{
				FieldName:     "columns",
				ConfigExtract: fmt.Sprintf(`{"version":%q,"tiles":[{"type":"EMPTY"}]}`, versions.CurrentVersion),
			},
		},
		{
			rawConfig: fmt.Sprintf(`{"version": %q, "columns": 0, "tiles": [{ "type": "EMPTY" }]}`, versions.CurrentVersion),
			errorID:   models.ConfigErrorInvalidFieldValue,
			errorData: models.ConfigErrorData{
				FieldName:     "columns",
				Expected:      "columns > 0",
				ConfigExtract: fmt.Sprintf(`{"version":%q,"columns":0,"tiles":[{"type":"EMPTY"}]}`, versions.CurrentVersion),
			},
		},
		{
			rawConfig: fmt.Sprintf(`{"version": %q, "columns": 1, "zoom": 0, "tiles": [{ "type": "EMPTY" }]}`, versions.CurrentVersion),
			errorID:   models.ConfigErrorInvalidFieldValue,
			errorData: models.ConfigErrorData{
				FieldName:     "zoom",
				Expected:      "zoom > 0",
				ConfigExtract: fmt.Sprintf(`{"version":%q,"columns":1,"zoom":0,"tiles":[{"type":"EMPTY"}]}`, versions.CurrentVersion),
			},
		},
		{
			rawConfig: fmt.Sprintf(`{"version": %q, "columns": 1, "zoom": 19.8, "tiles": [{ "type": "EMPTY" }]}`, versions.CurrentVersion),
			errorID:   models.ConfigErrorInvalidFieldValue,
			errorData: models.ConfigErrorData{
				FieldName:     "zoom",
				Expected:      "zoom <= 10",
				ConfigExtract: fmt.Sprintf(`{"version":%q,"columns":1,"zoom":19.8,"tiles":[{"type":"EMPTY"}]}`, versions.CurrentVersion),
			},
		},
		{
			rawConfig: fmt.Sprintf(`{"version": %q, "columns": 1}`, versions.CurrentVersion),
			errorID:   models.ConfigErrorMissingRequiredField,
			errorData: models.ConfigErrorData{
				FieldName:     "tiles",
				ConfigExtract: fmt.Sprintf(`{"version":%q,"columns":1}`, versions.CurrentVersion),
			},
		},
		{
			rawConfig: fmt.Sprintf(`{"version": %q, "columns": 1, "tiles": []}`, versions.CurrentVersion),
			errorID:   models.ConfigErrorInvalidFieldValue,
			errorData: models.ConfigErrorData{
				FieldName:     "tiles",
				ConfigExtract: fmt.Sprintf(`{"version":%q,"columns":1,"tiles":[]}`, versions.CurrentVersion),
			},
		},
	} {
		conf, err := readConfig(testcase.rawConfig)
		if assert.NoError(t, err) {
			usecase := initConfigUsecase(nil)
			usecase.Verify(conf)
			if assert.Len(t, conf.Errors, 1) {
				assert.Equal(t, testcase.errorID, conf.Errors[0].ID)
				assert.Equal(t, testcase.errorData, conf.Errors[0].Data)
			}
		}
	}
}

func TestUsecase_VerifyTile_Success(t *testing.T) {
	rawConfig := `{ "type": "PORT", "columnSpan": 2, "rowSpan": 2, "params": { "hostname": "bserver.com", "port": 22 } }`

	tile, conf := initConfig(t, rawConfig)
	usecase := initConfigUsecase(nil)
	usecase.verifyTile(conf, tile, nil)

	assert.Len(t, conf.Errors, 0)
}

func TestUsecase_VerifyTile_Success_Empty(t *testing.T) {
	rawConfig := `{ "type": "EMPTY" }`

	tile, conf := initConfig(t, rawConfig)
	usecase := initConfigUsecase(nil)
	usecase.verifyTile(conf, tile, nil)

	assert.Len(t, conf.Errors, 0)
}

func TestUsecase_VerifyTile_Success_Group(t *testing.T) {
	rawConfig := `
      { "type": "GROUP", "label": "...", "tiles": [
          { "type": "PING", "params": { "hostname": "aserver.com" } },
          { "type": "PORT", "params": { "hostname": "bserver.com", "port": 22 } }
			]}
`

	tile, conf := initConfig(t, rawConfig)
	usecase := initConfigUsecase(nil)
	usecase.verifyTile(conf, tile, nil)

	assert.Len(t, conf.Errors, 0)
}

func TestUsecase_VerifyTile_Failed(t *testing.T) {
	for _, testcase := range []struct {
		rawConfig string
		errorID   models.ConfigErrorID
		errorData models.ConfigErrorData
	}{
		{
			rawConfig: `{ "type": "PING", "columnSpan": -1, "params": { "hostname": "server.com" } }`,
			errorID:   models.ConfigErrorInvalidFieldValue,
			errorData: models.ConfigErrorData{
				FieldName:     "columnSpan",
				Expected:      "columnSpan > 0",
				ConfigExtract: `{"type":"PING","columnSpan":-1,"params":{"hostname":"server.com"}}`,
			},
		},
		{
			rawConfig: `{ "type": "PING", "rowSpan": -1, "params": { "hostname": "server.com" } }`,
			errorID:   models.ConfigErrorInvalidFieldValue,
			errorData: models.ConfigErrorData{
				FieldName:     "rowSpan",
				Expected:      "rowSpan > 0",
				ConfigExtract: `{"type":"PING","rowSpan":-1,"params":{"hostname":"server.com"}}`,
			},
		},
		{
			rawConfig: `
					{ "type": "GROUP", "tiles": [
							{ "type": "EMPTY" }
					]}
		`,
			errorID: models.ConfigErrorUnauthorizedSubtileType,
			errorData: models.ConfigErrorData{
				ConfigExtract:          `{"type":"GROUP","tiles":[{"type":"EMPTY"}]}`,
				ConfigExtractHighlight: `{"type":"EMPTY"}`,
			},
		},
		{
			rawConfig: `
					{ "type": "GROUP", "tiles": [
							{ "type": "GROUP" }
					]}
		`,
			errorID: models.ConfigErrorUnauthorizedSubtileType,
			errorData: models.ConfigErrorData{
				ConfigExtract:          `{"type":"GROUP","tiles":[{"type":"GROUP"}]}`,
				ConfigExtractHighlight: `{"type":"GROUP"}`,
			},
		},
		{
			rawConfig: `{ "type": "GROUP", "params": {"test": "test"}}`,
			errorID:   models.ConfigErrorUnauthorizedField,
			errorData: models.ConfigErrorData{
				FieldName:     "params",
				ConfigExtract: `{"type":"GROUP","params":{"test":"test"}}`,
			},
		},
		{
			rawConfig: `{ "type": "GROUP"}`,
			errorID:   models.ConfigErrorMissingRequiredField,
			errorData: models.ConfigErrorData{
				FieldName:     "tiles",
				ConfigExtract: `{"type":"GROUP"}`,
			},
		},
		{
			rawConfig: `{ "type": "GROUP", "tiles": []}`,
			errorID:   models.ConfigErrorInvalidFieldValue,
			errorData: models.ConfigErrorData{
				FieldName:     "tiles",
				ConfigExtract: `{"type":"GROUP"}`,
			},
		},
		{
			rawConfig: `{ "type": "PING" }`,
			errorID:   models.ConfigErrorMissingRequiredField,
			errorData: models.ConfigErrorData{
				FieldName:     "params",
				ConfigExtract: `{"type":"PING","configVariant":"default"}`,
			},
		},
		{
			rawConfig: `{ "type": "PING", "params": { } }`,
			errorID:   models.ConfigErrorMissingRequiredField,
			errorData: models.ConfigErrorData{
				FieldName:     "hostname",
				ConfigExtract: `{"type":"PING","configVariant":"default","params":{}}`,
			},
		},
		{
			rawConfig: `{ "type": "PING", "params": { "host": "server.com" } }`,
			errorID:   models.ConfigErrorUnknownField,
			errorData: models.ConfigErrorData{
				FieldName:     "host",
				ConfigExtract: `{"type":"PING","params":{"host":"server.com"},"configVariant":"default"}`,
				Expected:      "hostname",
			},
		},
		{
			rawConfig: `{ "type": "PORT", "params": { "hostname": "server.com", "port": -20 } }`,
			errorID:   models.ConfigErrorInvalidFieldValue,
			errorData: models.ConfigErrorData{
				FieldName:     "port",
				ConfigExtract: `{"type":"PORT","params":{"hostname":"server.com","port":-20},"configVariant":"default"}`,
				Expected:      "port > 0",
			},
		},
		{
			rawConfig: `{ "type": "PING", "params": { "hostname": ["server.com"] } }`,
			errorID:   models.ConfigErrorUnexpectedError,
			errorData: models.ConfigErrorData{
				FieldName:     "params",
				ConfigExtract: `{"type":"PING","params":{"hostname":["server.com"]},"configVariant":"default"}`,
			},
		},
		{
			rawConfig: `{ "type": "JENKINS-BUILD", "configVariant": "disabledVariant", "params": { } }`,
			errorID:   models.ConfigErrorDisabledVariant,
			errorData: models.ConfigErrorData{
				FieldName:     "configVariant",
				Value:         `"disabledVariant"`,
				ConfigExtract: `{"type":"JENKINS-BUILD","configVariant":"disabledVariant"}`,
			},
		},
	} {
		tile, conf := initConfig(t, testcase.rawConfig)
		usecase := initConfigUsecase(nil)
		usecase.verifyTile(conf, tile, nil)

		if assert.Len(t, conf.Errors, 1) {
			assert.Equal(t, testcase.errorID, conf.Errors[0].ID)
			assert.Equal(t, testcase.errorData, conf.Errors[0].Data)
		}
	}
}

func TestUsecase_VerifyTile_Failed_WrongMinimalVerison(t *testing.T) {
	rawConfig := `{ "type": "PING", "params": { "hostname": "server.com" } }`

	tile, conf := initConfig(t, rawConfig)
	usecase := initConfigUsecase(nil)
	usecase.registry.TileMetadata["PING"].MinimalVersion = "999.0"
	usecase.verifyTile(conf, tile, nil)

	if assert.Len(t, conf.Errors, 1) {
		assert.Equal(t, models.ConfigErrorUnsupportedTileInThisVersion, conf.Errors[0].ID)
		assert.Equal(t, "type", conf.Errors[0].Data.FieldName)
		assert.Equal(t, `{"type":"PING","params":{"hostname":"server.com"},"configVariant":"default"}`, conf.Errors[0].Data.ConfigExtract)
		assert.Equal(t, `version >= "999.0"`, conf.Errors[0].Data.Expected)
	}
}

func TestUsecase_VerifyTile_Failed_WrongTileType(t *testing.T) {
	rawConfig := `{ "type": "PONG", "params": { "hostname": "server.com" } }`

	tile, conf := initConfig(t, rawConfig)
	usecase := initConfigUsecase(nil)
	usecase.verifyTile(conf, tile, nil)

	if assert.Len(t, conf.Errors, 1) {
		assert.Equal(t, models.ConfigErrorUnknownTileType, conf.Errors[0].ID)
		assert.Equal(t, "type", conf.Errors[0].Data.FieldName)
		assert.Equal(t, `{"type":"PONG","params":{"hostname":"server.com"},"configVariant":"default"}`, conf.Errors[0].Data.ConfigExtract)
	}
}

func TestUsecase_VerifyTile_WithGenerator(t *testing.T) {
	rawConfig := `{ "type": "GENERATE:JENKINS-BUILD", "configVariant": "default", "params": { "job": "job1" } }`

	tile, conf := initConfig(t, rawConfig)
	params := &jenkinsModels.BuildParams{Job: "test"}
	mockBuilder := func(_ interface{}) ([]models.GeneratedTile, error) {
		return []models.GeneratedTile{{Params: params}}, nil
	}

	usecase := initConfigUsecase(nil)
	usecase.registry.RegisterGenerator(jenkinsApi.JenkinsBuildTileType, versions.MinimalVersion, []coreModels.VariantName{coreModels.DefaultVariantName}).
		Enable(coreModels.DefaultVariantName, &jenkinsModels.BuildGeneratorParams{}, mockBuilder)
	usecase.verifyTile(conf, tile, nil)

	assert.Len(t, conf.Errors, 0)
}

func TestUsecase_VerifyTile_WithGenerator_WithWrongGenerator(t *testing.T) {
	rawConfig := `{ "type": "GENERATE:PING", "params": {}}`

	tile, conf := initConfig(t, rawConfig)
	usecase := initConfigUsecase(nil)
	usecase.registry.RegisterGenerator(jenkinsApi.JenkinsBuildTileType, versions.MinimalVersion, []coreModels.VariantName{coreModels.DefaultVariantName}).
		Enable(coreModels.DefaultVariantName, &jenkinsModels.BuildGeneratorParams{}, nil)

	usecase.verifyTile(conf, tile, nil)

	if assert.Len(t, conf.Errors, 1) {
		assert.Equal(t, models.ConfigErrorUnknownGeneratorTileType, conf.Errors[0].ID)
		assert.Equal(t, "type", conf.Errors[0].Data.FieldName)
		assert.Equal(t, `{"type":"GENERATE:PING","configVariant":"default"}`, conf.Errors[0].Data.ConfigExtract)
		assert.Equal(t, `GENERATE:JENKINS-BUILD`, conf.Errors[0].Data.Expected)
	}
}

func TestUsecase_VerifyTile_WithWrongVariant(t *testing.T) {
	rawConfig := `{ "type": "JENKINS-BUILD", "configVariant": "test", "params": { "job": "job1" } }`

	tile, conf := initConfig(t, rawConfig)

	usecase := initConfigUsecase(nil)
	usecase.verifyTile(conf, tile, nil)

	if assert.Len(t, conf.Errors, 1) {
		assert.Equal(t, models.ConfigErrorUnknownVariant, conf.Errors[0].ID)
		assert.Equal(t, "configVariant", conf.Errors[0].Data.FieldName)
		assert.Equal(t, `"test"`, conf.Errors[0].Data.Value)
		assert.Contains(t, conf.Errors[0].Data.Expected, coreModels.DefaultVariantName)
		assert.Contains(t, conf.Errors[0].Data.Expected, "disabledVariant")
		assert.Equal(t, `{"type":"JENKINS-BUILD","params":{"job":"job1"},"configVariant":"test"}`, conf.Errors[0].Data.ConfigExtract)
	}
}

func TestUsecase_VerifyTile_WithGenerator_WithWrongVariant(t *testing.T) {
	rawConfig := `{ "type": "GENERATE:JENKINS-BUILD", "configVariant": "test", "params": { "job": "job1" } }`

	tile, conf := initConfig(t, rawConfig)
	params := &jenkinsModels.BuildParams{Job: "test"}
	mockBuilder := func(_ interface{}) ([]models.GeneratedTile, error) {
		return []models.GeneratedTile{{Params: params}}, nil
	}

	usecase := initConfigUsecase(nil)
	usecase.registry.RegisterGenerator(jenkinsApi.JenkinsBuildTileType, versions.MinimalVersion, []coreModels.VariantName{coreModels.DefaultVariantName}).
		Enable(coreModels.DefaultVariantName, &jenkinsModels.BuildGeneratorParams{}, mockBuilder)
	usecase.verifyTile(conf, tile, nil)

	if assert.Len(t, conf.Errors, 1) {
		assert.Equal(t, models.ConfigErrorUnknownVariant, conf.Errors[0].ID)
		assert.Equal(t, "configVariant", conf.Errors[0].Data.FieldName)
		assert.Equal(t, `"test"`, conf.Errors[0].Data.Value)
		assert.Contains(t, conf.Errors[0].Data.Expected, coreModels.DefaultVariantName)
		assert.Equal(t, `{"type":"GENERATE:JENKINS-BUILD","params":{"job":"job1"},"configVariant":"test"}`, conf.Errors[0].Data.ConfigExtract)
	}
}

type minimalVersionTest struct {
	Field1 string `json:"field1" available:"since=999.0"`
}

func (s *minimalVersionTest) Validate() []validator.Error { return nil }

func TestUsecase_VerifyTile_FieldMinimalVersion(t *testing.T) {
	rawConfig := `{ "type": "TEST", "params": { "field1": "server.com" } }`
	tile, conf := initConfig(t, rawConfig)
	usecase := initConfigUsecase(nil)
	usecase.registry.RegisterTile("TEST", versions.MinimalVersion, []coreModels.VariantName{coreModels.DefaultVariantName}).
		Enable(coreModels.DefaultVariantName, &minimalVersionTest{}, "/test/default/test")

	usecase.verifyTile(conf, tile, nil)

	if assert.Len(t, conf.Errors, 1) {
		assert.Equal(t, models.ConfigErrorUnsupportedTileParamInThisVersion, conf.Errors[0].ID)
		assert.Equal(t, "field1", conf.Errors[0].Data.FieldName)
		assert.Equal(t, "version >= 999.0", conf.Errors[0].Data.Expected)
		assert.Equal(t, `{"type":"TEST","params":{"field1":"server.com"},"configVariant":"default"}`, conf.Errors[0].Data.ConfigExtract)
	}
}
