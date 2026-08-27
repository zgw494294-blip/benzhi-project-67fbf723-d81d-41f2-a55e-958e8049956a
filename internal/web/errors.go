package web

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"seed-germination-workbench/internal/domain"
	"seed-germination-workbench/internal/store"
)

func errJSON(response http.ResponseWriter, err error) {
	status := http.StatusBadRequest
	body := map[string]any{"error": err.Error()}
	var conflict *store.ConflictError
	if errors.As(err, &conflict) {
		status = http.StatusConflict
		body["conflict"] = conflict
	}
	var validation *domain.ValidationErrors
	if errors.As(err, &validation) {
		status = http.StatusUnprocessableEntity
		body["issues"] = validation.Issues
	}
	var batch *domain.ObservationBatchError
	if errors.As(err, &batch) {
		status = http.StatusUnprocessableEntity
		body["missingReplicates"] = batch.MissingReplicates
		body["remainingByGroup"] = batch.RemainingByGroup
		body["issues"] = batch.Issues
	}
	if strings.Contains(err.Error(), "版本冲突") || strings.Contains(err.Error(), "摘要不匹配") || strings.Contains(err.Error(), "数据完整性错误") {
		status = http.StatusConflict
	}
	response.Header().Set("X-Error-Code", strconv.Itoa(status))
	writeJSON(response, status, body)
	log.Print(err)
}
