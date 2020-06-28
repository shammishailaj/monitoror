//+build faker

package monitorable

import coreModels "github.com/monitoror/monitoror/models"

type DefaultMonitorableFaker struct {
}

func (m *DefaultMonitorableFaker) GetVariantsNames() []coreModels.VariantName {
	return []coreModels.VariantName{coreModels.DefaultVariantName}
}

func (m *DefaultMonitorableFaker) Validate(_ coreModels.VariantName) (bool, []error) {
	return true, nil
}
