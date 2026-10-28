package application

import (
	"testing"

	"jimu/internal/capabilities/tenant/domain"

	"github.com/stretchr/testify/assert"
)

func TestTenantResponseSerializesPlanIDAsString(t *testing.T) {
	resp := ToTenantResponse(domain.Tenant{PlanID: 373398409660334080})

	assert.Equal(t, "373398409660334080", resp.PlanID)
}
