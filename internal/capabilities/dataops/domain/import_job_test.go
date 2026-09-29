package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestImportJobTableNameAndStatus(t *testing.T) {
	assert.Equal(t, "import_jobs", (ImportJob{}).TableName())
	assert.Equal(t, ImportJobStatus("pending"), ImportJobPending)
	assert.Equal(t, ImportJobStatus("processing"), ImportJobProcessing)
	assert.Equal(t, ImportJobStatus("completed"), ImportJobCompleted)
	assert.Equal(t, ImportJobStatus("failed"), ImportJobFailed)
}
