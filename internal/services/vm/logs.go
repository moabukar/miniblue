package vm

import (
	"bufio"
	"fmt"
	"net/http"
	"strconv"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/moabukar/miniblue/internal/azerr"
)

// ---------------------------------------------------------------------------
// Logs
//
// GET .../virtualMachines/{vm}/logs?service=<name>&tail=<n>&follow=<bool>
//
// With service= the response is that container's output. Without it, the
// combined view: provisioning events first, then each service's output in
// labeled sections (follow mode multiplexes lines with a [service] prefix).
// ---------------------------------------------------------------------------

func (h *Handler) GetLogs(w http.ResponseWriter, r *http.Request) {
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

	svcFilter := r.URL.Query().Get("service")
	follow := r.URL.Query().Get("follow") == "true"
	tail := 0
	if t := r.URL.Query().Get("tail"); t != "" {
		n, err := strconv.Atoi(t)
		if err != nil || n < 0 {
			azerr.BadRequest(w, "tail must be a non-negative integer")
			return
		}
		tail = n
	}

	type target struct {
		name      string
		container string
	}
	var targets []target
	if svcFilter != "" {
		v, ok := h.store.Get(svcKey(sub, rg, vmName, svcFilter))
		if !ok {
			azerr.NotFound(w, "Microsoft.Compute/virtualMachines/services", svcFilter)
			return
		}
		rec, _ := v.(map[string]interface{})
		cn, _ := rec["_containerName"].(string)
		targets = append(targets, target{svcFilter, cn})
	} else {
		for _, rec := range h.serviceRecords(sub, rg, vmName) {
			name, _ := rec["name"].(string)
			cn, _ := rec["_containerName"].(string)
			targets = append(targets, target{name, cn})
		}
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

	if !follow {
		// Snapshot: provisioning events (combined view only) + per-target logs.
		if svcFilter == "" {
			for _, line := range provisionEvents(vmRec) {
				fmt.Fprintln(w, line)
			}
		}
		for _, t := range targets {
			if svcFilter == "" {
				fmt.Fprintf(w, "==> service: %s <==\n", t.name)
			}
			out, err := dockerLogsOutput(t.container, tail)
			if err != nil {
				fmt.Fprintf(w, "(logs unavailable: %v)\n", err)
				continue
			}
			fmt.Fprint(w, out)
		}
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		azerr.WriteError(w, http.StatusInternalServerError, "StreamingUnsupported", "Streaming is not supported by this connection.")
		return
	}

	var mu sync.Mutex
	writeLine := func(line string) {
		mu.Lock()
		defer mu.Unlock()
		fmt.Fprintln(w, line)
		flusher.Flush()
	}

	if svcFilter == "" {
		for _, line := range provisionEvents(vmRec) {
			writeLine(line)
		}
	}

	ctx := r.Context()
	var wg sync.WaitGroup
	for _, t := range targets {
		reader, stop, err := dockerLogsFollow(t.container, tail)
		if err != nil {
			writeLine(fmt.Sprintf("(logs unavailable for %s: %v)", t.name, err))
			continue
		}
		// Kill the docker logs process as soon as the client disconnects.
		go func() {
			<-ctx.Done()
			stop()
		}()
		wg.Add(1)
		go func(name string, combined bool) {
			defer wg.Done()
			defer reader.Close()
			scanner := bufio.NewScanner(reader)
			scanner.Buffer(make([]byte, 64*1024), 1024*1024)
			for scanner.Scan() {
				if combined {
					writeLine("[" + name + "] " + scanner.Text())
				} else {
					writeLine(scanner.Text())
				}
			}
		}(t.name, svcFilter == "")
	}
	wg.Wait()
}
