package vm

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/moabukar/miniblue/internal/azerr"
)

// RunCommand mirrors Azure's virtualMachines runCommand action: the script is
// executed inside the VM container through `sh -c`, so shell semantics (env
// expansion, pipes) apply. The combined output and exit status are returned
// in the standard {"value":[{"code","message"}]} envelope.
func (h *Handler) RunCommand(w http.ResponseWriter, r *http.Request) {
	sub := chi.URLParam(r, "subscriptionId")
	rg := chi.URLParam(r, "resourceGroupName")
	vmName := chi.URLParam(r, "vmName")

	vmRec, ok := h.getVM(sub, rg, vmName)
	if !ok {
		azerr.NotFound(w, "Microsoft.Compute/virtualMachines", vmName)
		return
	}
	if !h.dockerAvail {
		dockerUnavailable(w)
		return
	}
	h.refreshVM(vmRec)
	if powerState(vmRec) != "running" {
		azerr.WriteError(w, http.StatusConflict, "VMNotRunning",
			"The virtual machine '"+vmName+"' must be running to execute commands.")
		return
	}

	var input struct {
		CommandID string   `json:"commandId"`
		Script    []string `json:"script"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		azerr.BadRequest(w, "Could not parse request body: "+err.Error())
		return
	}
	if len(input.Script) == 0 {
		azerr.BadRequest(w, "script must contain at least one line")
		return
	}

	output, exitCode, err := dockerExec(vmContainerName(rg, vmName), strings.Join(input.Script, "\n"))
	if err != nil {
		azerr.WriteError(w, http.StatusConflict, "ContainerStartFailed", err.Error())
		return
	}

	code := "ProvisioningState/succeeded"
	if exitCode != 0 {
		code = fmt.Sprintf("ProvisioningState/failed/exitCode=%d", exitCode)
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"value": []interface{}{
			map[string]interface{}{
				"code":    code,
				"level":   "Info",
				"message": output,
			},
		},
	})
}
