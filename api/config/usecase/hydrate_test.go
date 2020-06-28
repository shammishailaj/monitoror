package usecase

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/monitoror/monitoror/api/config/models"
	"github.com/monitoror/monitoror/api/config/versions"
	coreModels "github.com/monitoror/monitoror/models"
	jenkinsApi "github.com/monitoror/monitoror/monitorables/jenkins/api"
	jenkinsModels "github.com/monitoror/monitoror/monitorables/jenkins/api/models"

	"github.com/stretchr/testify/assert"
)

func TestUsecase_Hydrate(t *testing.T) {
	input := `
{
  "columns": 4,
  "tiles": [
    { "type": "EMPTY" },
    { "type": "PING", "params": { "hostname": "aserver.com", "values": [123, 456] } },
    { "type": "PORT", "params": { "hostname": "bserver.com", "port": 22 } },
    { "type": "GROUP", "label": "...", "tiles": [
      { "type": "PING", "params": { "hostname": "aserver.com" } },
      { "type": "PORT", "params": { "hostname": "bserver.com", "port": 22 } }
    ]},
		{ "type": "JENKINS-BUILD", "params": { "job": "test" } },
		{ "type": "JENKINS-BUILD", "configVariant": "variant1", "params": { "job": "test" } },
    { "type": "PINGDOM-CHECK", "params": { "id": 10000000 } }
  ]
}
`

	usecase := initConfigUsecase(nil)
	jenkinsTileEnabler := usecase.registry.RegisterTile(jenkinsApi.JenkinsBuildTileType, versions.MinimalVersion, []coreModels.VariantName{coreModels.DefaultVariantName, "variant1"})
	jenkinsTileEnabler.Enable(coreModels.DefaultVariantName, &jenkinsModels.BuildParams{}, "/jenkins/default/build")
	jenkinsTileEnabler.Enable("variant1", &jenkinsModels.BuildParams{}, "/jenkins/variant1/build")

	config, err := readConfig(input)

	assert.NoError(t, err)

	usecase.Hydrate(config)
	assert.Len(t, config.Errors, 0)

	assert.Equal(t, "/ping/default/ping?hostname=aserver.com&values=123&values=456", config.Config.Tiles[1].URL)
	assert.Equal(t, 1000, *config.Config.Tiles[1].InitialMaxDelay)
	assert.Equal(t, "/port/default/port?hostname=bserver.com&port=22", config.Config.Tiles[2].URL)
	assert.Equal(t, 1000, *config.Config.Tiles[2].InitialMaxDelay)

	group := config.Config.Tiles[3].Tiles
	assert.Equal(t, "/ping/default/ping?hostname=aserver.com", group[0].URL)
	assert.Equal(t, 1000, *group[0].InitialMaxDelay)
	assert.Equal(t, "/port/default/port?hostname=bserver.com&port=22", group[1].URL)
	assert.Equal(t, 1000, *group[1].InitialMaxDelay)

	assert.Equal(t, "/jenkins/default/build?job=test", config.Config.Tiles[4].URL)
	assert.Equal(t, 1000, *config.Config.Tiles[4].InitialMaxDelay)
	assert.Equal(t, "/jenkins/variant1/build?job=test", config.Config.Tiles[5].URL)
	assert.Equal(t, 1000, *config.Config.Tiles[5].InitialMaxDelay)
	assert.Equal(t, "/pingdom/default/check?id=10000000", config.Config.Tiles[6].URL)
	assert.Equal(t, 1000, *config.Config.Tiles[6].InitialMaxDelay)
}

func TestUsecase_Hydrate_WithGenerator(t *testing.T) {
	input := `
{
  "columns": 4,
  "tiles": [
    { "type": "GENERATE:JENKINS-BUILD", "params": {"job": "test"}},
    { "type": "GROUP", "label": "...", "tiles": [
      { "type": "PING", "params": { "hostname": "aserver.com" } },
			{ "type": "GENERATE:JENKINS-BUILD", "params": {"job": "test"}}
    ]},
    { "type": "GROUP", "label": "...", "tiles": [
    	{ "type": "GENERATE:JENKINS-BUILD", "params": {"job": "test"}}
    ]},
    { "type": "GROUP", "label": "...", "tiles": [
    	{ "type": "GENERATE:JENKINS-BUILD", "label": "Test Label", "params": {"job": "test"}}
    ]}
  ]
}
`
	params := &jenkinsModels.BuildParams{Job: "test"}
	mockBuilder := func(_ interface{}) ([]models.GeneratedTile, error) {
		return []models.GeneratedTile{{Params: params}}, nil
	}

	usecase := initConfigUsecase(nil)
	usecase.registry.RegisterGenerator(jenkinsApi.JenkinsBuildTileType, versions.MinimalVersion, []coreModels.VariantName{coreModels.DefaultVariantName}).
		Enable(coreModels.DefaultVariantName, &jenkinsModels.BuildGeneratorParams{}, mockBuilder)

	config, err := readConfig(input)
	assert.NoError(t, err)

	usecase.Hydrate(config)
	assert.Len(t, config.Errors, 0)

	assert.Equal(t, 4, len(config.Config.Tiles))
	assert.Equal(t, jenkinsApi.JenkinsBuildTileType, config.Config.Tiles[0].Type)
	assert.Equal(t, "/jenkins/default/build?job=test", config.Config.Tiles[0].URL)
	assert.Equal(t, 1000, *config.Config.Tiles[0].InitialMaxDelay)
	assert.Equal(t, "", config.Config.Tiles[2].Tiles[0].Label)
	assert.Equal(t, jenkinsApi.JenkinsBuildTileType, config.Config.Tiles[1].Tiles[1].Type)
	assert.Equal(t, "/jenkins/default/build?job=test", config.Config.Tiles[1].Tiles[1].URL)
	assert.Equal(t, 1000, *config.Config.Tiles[1].Tiles[1].InitialMaxDelay)
	assert.Equal(t, "", config.Config.Tiles[2].Tiles[0].Label)
	assert.Equal(t, jenkinsApi.JenkinsBuildTileType, config.Config.Tiles[2].Tiles[0].Type)
	assert.Equal(t, "/jenkins/default/build?job=test", config.Config.Tiles[2].Tiles[0].URL)
	assert.Equal(t, 1000, *config.Config.Tiles[2].Tiles[0].InitialMaxDelay)
	assert.Equal(t, "", config.Config.Tiles[2].Tiles[0].Label)
	assert.Equal(t, jenkinsApi.JenkinsBuildTileType, config.Config.Tiles[3].Tiles[0].Type)
	assert.Equal(t, "/jenkins/default/build?job=test", config.Config.Tiles[3].Tiles[0].URL)
	assert.Equal(t, 1000, *config.Config.Tiles[3].Tiles[0].InitialMaxDelay)
	assert.Equal(t, "Test Label", config.Config.Tiles[3].Tiles[0].Label)
}

func TestUsecase_Hydrate_WithGeneratorEmpty(t *testing.T) {
	input := `
{
  "columns": 4,
  "tiles": [
    { "type": "PING", "params": { "hostname": "aserver.com" } },
    { "type": "GROUP", "label": "...", "tiles": [
    	{ "type": "GENERATE:JENKINS-BUILD", "params": {"job": "test"}}
    ]},
    { "type": "PING", "params": { "hostname": "bserver.com" } }
  ]
}
`
	mockBuilder := func(_ interface{}) ([]models.GeneratedTile, error) { return []models.GeneratedTile{}, nil }

	usecase := initConfigUsecase(nil)
	usecase.registry.RegisterGenerator(jenkinsApi.JenkinsBuildTileType, versions.MinimalVersion, []coreModels.VariantName{coreModels.DefaultVariantName}).
		Enable(coreModels.DefaultVariantName, &jenkinsModels.BuildGeneratorParams{}, mockBuilder)

	config, err := readConfig(input)
	assert.NoError(t, err)

	usecase.Hydrate(config)
	assert.Len(t, config.Errors, 0)

	assert.Equal(t, 2, len(config.Config.Tiles))
}

func TestUsecase_Hydrate_WithGenerator_WithError(t *testing.T) {
	input := `
{
  "columns": 4,
  "tiles": [
    { "type": "GENERATE:JENKINS-BUILD", "params": {"job": "test"}},
    { "type": "GROUP", "label": "...", "tiles": [
      { "type": "PING", "params": { "hostname": "aserver.com" } },
			{ "type": "GENERATE:JENKINS-BUILD", "params": {"job": "test"}}
    ]},
    { "type": "GROUP", "label": "...", "tiles": [
    	{ "type": "GENERATE:JENKINS-BUILD", "configVariant": "variant1", "params": {"job": "test"}}
    ]}
  ]
}
`
	params := &jenkinsModels.BuildParams{Job: "test"}
	mockBuilder := func(_ interface{}) ([]models.GeneratedTile, error) {
		return []models.GeneratedTile{{Params: params}}, nil
	}
	mockBuilder2 := func(_ interface{}) ([]models.GeneratedTile, error) { return nil, errors.New("unable to find job") }

	usecase := initConfigUsecase(nil)
	tileGeneratorEnabler := usecase.registry.RegisterGenerator(jenkinsApi.JenkinsBuildTileType, versions.MinimalVersion, []coreModels.VariantName{coreModels.DefaultVariantName, "variant1"})
	tileGeneratorEnabler.Enable(coreModels.DefaultVariantName, &jenkinsModels.BuildGeneratorParams{}, mockBuilder)
	tileGeneratorEnabler.Enable("variant1", &jenkinsModels.BuildGeneratorParams{}, mockBuilder2)

	config, err := readConfig(input)
	assert.NoError(t, err)

	usecase.Hydrate(config)
	assert.Len(t, config.Errors, 1)
	assert.Equal(t, config.Errors[0].ID, models.ConfigErrorUnableToHydrate)
	assert.Contains(t, config.Errors[0].Data.ConfigExtract, `GENERATE:JENKINS-BUILD`)
}

func TestUsecase_Hydrate_WithGenerator_WithTimeoutError(t *testing.T) {
	input := `
{
  "columns": 4,
  "tiles": [
    { "type": "GENERATE:JENKINS-BUILD", "params": {"job": "test"}},
    { "type": "GROUP", "label": "...", "tiles": [
      { "type": "PING", "params": { "hostname": "aserver.com" } },
			{ "type": "GENERATE:JENKINS-BUILD", "params": {"job": "test"}}
    ]},
    { "type": "GROUP", "label": "...", "tiles": [
    	{ "type": "GENERATE:JENKINS-BUILD", "configVariant": "variant1", "params": {"job": "test"}}
    ]}
  ]
}
`
	params := &jenkinsModels.BuildParams{Job: "test"}
	mockBuilder := func(_ interface{}) ([]models.GeneratedTile, error) {
		return []models.GeneratedTile{{Params: params}}, nil
	}
	mockBuilder2 := func(_ interface{}) ([]models.GeneratedTile, error) { return nil, context.DeadlineExceeded }

	usecase := initConfigUsecase(nil)
	tileGeneratorEnabler := usecase.registry.RegisterGenerator(jenkinsApi.JenkinsBuildTileType, versions.MinimalVersion, []coreModels.VariantName{coreModels.DefaultVariantName, "variant1"})
	tileGeneratorEnabler.Enable(coreModels.DefaultVariantName, &jenkinsModels.BuildGeneratorParams{}, mockBuilder)
	tileGeneratorEnabler.Enable("variant1", &jenkinsModels.BuildGeneratorParams{}, mockBuilder2)
	config, err := readConfig(input)
	assert.NoError(t, err)

	usecase.Hydrate(config)
	assert.Len(t, config.Errors, 1)
	assert.Equal(t, config.Errors[0].ID, models.ConfigErrorUnableToHydrate)
	assert.Contains(t, config.Errors[0].Data.ConfigExtract, `GENERATE:JENKINS-BUILD`)
}

func TestUsecase_Hydrate_WithGenerator_WithTimeoutCache(t *testing.T) {
	input := `
{
  "columns": 4,
  "tiles": [
    { "type": "GENERATE:JENKINS-BUILD", "params": {"job": "test"}}
	]
}
`
	usecase := initConfigUsecase(nil)

	params := &jenkinsModels.BuildParams{Job: "test"}
	cachedResult := []models.GeneratedTile{{Params: params}}
	cacheKey := fmt.Sprintf("%s:%s_%s_%s", TileGeneratorStoreKeyPrefix, "GENERATE:JENKINS-BUILD", "default", `{"job":"test"}`)
	_ = usecase.generatorTileStore.Add(cacheKey, cachedResult, 0)

	mockBuilder := func(_ interface{}) ([]models.GeneratedTile, error) { return nil, context.DeadlineExceeded }
	usecase.registry.RegisterGenerator(jenkinsApi.JenkinsBuildTileType, versions.MinimalVersion, []coreModels.VariantName{coreModels.DefaultVariantName}).
		Enable(coreModels.DefaultVariantName, &jenkinsModels.BuildGeneratorParams{}, mockBuilder)

	config, err := readConfig(input)
	if assert.NoError(t, err) {
		usecase.Hydrate(config)
		assert.Len(t, config.Errors, 0)
		assert.Equal(t, jenkinsApi.JenkinsBuildTileType, config.Config.Tiles[0].Type)
		assert.Equal(t, "/jenkins/default/build?job=test", config.Config.Tiles[0].URL)
	}
}

func TestUsecase_Hydrate_TwoGenerators(t *testing.T) {
	input := `
{
  "columns": 4,
  "tiles": [
    { "type": "GENERATE:JENKINS-BUILD", "params": {"job": "test"}},
    { "type": "GENERATE:JENKINS-BUILD", "params": {"job": "test2"}}
  ]
}
`
	mockBuilder := func(_ interface{}) ([]models.GeneratedTile, error) {
		return []models.GeneratedTile{
			{Params: &jenkinsModels.BuildParams{Job: "test1"}},
			{Params: &jenkinsModels.BuildParams{Job: "test2"}},
		}, nil
	}

	usecase := initConfigUsecase(nil)
	usecase.registry.RegisterGenerator(jenkinsApi.JenkinsBuildTileType, versions.MinimalVersion, []coreModels.VariantName{coreModels.DefaultVariantName}).
		Enable(coreModels.DefaultVariantName, &jenkinsModels.BuildGeneratorParams{}, mockBuilder)
	config, err := readConfig(input)
	assert.NoError(t, err)

	usecase.Hydrate(config)
	assert.Len(t, config.Errors, 0)

	if assert.Equal(t, 4, len(config.Config.Tiles)) {
		assert.Equal(t, "/jenkins/default/build?job=test1", config.Config.Tiles[0].URL)
		assert.Equal(t, "/jenkins/default/build?job=test2", config.Config.Tiles[1].URL)
		assert.Equal(t, "/jenkins/default/build?job=test1", config.Config.Tiles[2].URL)
		assert.Equal(t, "/jenkins/default/build?job=test2", config.Config.Tiles[3].URL)
	}

}
