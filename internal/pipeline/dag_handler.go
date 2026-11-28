package pipeline

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/vulan1999/go-dag/internal/dag"
	"github.com/vulan1999/go-dag/internal/stage"
	"github.com/vulan1999/go-dag/web/components"
	"github.com/vulan1999/go-dag/web/pages"
)

func buildDAGView(p *Pipeline) components.DAGView {
	d := dag.New()
	nodeMap := make(map[string]Node, len(p.Nodes))
	var buildErr error

	for _, n := range p.Nodes {
		nodeMap[n.ID] = n
		if err := d.AddNode(n.ID); err != nil && buildErr == nil {
			buildErr = err
		}
	}

	for _, e := range p.Edges {
		if err := d.AddEdge(e.From, e.To); err != nil && buildErr == nil {
			buildErr = err
		}
	}

	levels, levelsErr := d.Levels()
	if buildErr == nil && levelsErr != nil {
		buildErr = levelsErr
	}

	toNodeView := func(id string) components.DAGNodeView {
		n := nodeMap[id]
		cfg := string(n.Config)
		if strings.TrimSpace(cfg) == "" {
			cfg = "{}"
		} else {
			var pretty json.RawMessage
			if json.Unmarshal(n.Config, &pretty) == nil {
				if formatted, err := json.MarshalIndent(pretty, "", "  "); err == nil {
					cfg = string(formatted)
				}
			}
		}
		return components.DAGNodeView{
			ID:         n.ID,
			Type:       n.Type,
			ConfigJSON: cfg,
			Columns:    extractNodeColumns(n),
			Parents:    d.Parents(n.ID),
			Children:   d.Children(n.ID),
		}
	}

	var waves [][]components.DAGNodeView
	if levelsErr == nil {
		waves = make([][]components.DAGNodeView, 0, len(levels))
		for _, lvl := range levels {
			wave := make([]components.DAGNodeView, 0, len(lvl))
			for _, id := range lvl {
				wave = append(wave, toNodeView(id))
			}
			waves = append(waves, wave)
		}
	} else if len(p.Nodes) > 0 {
		// Fallback single wave so the user can still inspect and fix invalid graphs.
		wave := make([]components.DAGNodeView, 0, len(p.Nodes))
		for _, n := range p.Nodes {
			wave = append(wave, toNodeView(n.ID))
		}
		waves = [][]components.DAGNodeView{wave}
	}

	flatNodes := make([]components.DAGNodeView, 0, len(p.Nodes))
	for _, id := range d.Nodes() {
		flatNodes = append(flatNodes, toNodeView(id))
	}

	edgeViews := make([]components.DAGEdgeView, 0, len(p.Edges))
	for _, e := range p.Edges {
		edgeViews = append(edgeViews, components.DAGEdgeView{
			From: e.From,
			To:   e.To,
		})
	}

	errMsg := ""
	if buildErr != nil {
		errMsg = buildErr.Error()
	}

	return components.DAGView{
		PipelineID:   p.ID,
		PipelineName: p.Name,
		Waves:        waves,
		Nodes:        flatNodes,
		Edges:        edgeViews,
		Valid:        buildErr == nil,
		ErrorMsg:     errMsg,
	}
}

func validatePipelineDAG(p *Pipeline) error {
	d := dag.New()
	for _, n := range p.Nodes {
		if err := d.AddNode(n.ID); err != nil {
			return err
		}
	}
	for _, e := range p.Edges {
		if err := d.AddEdge(e.From, e.To); err != nil {
			return err
		}
	}
	return d.Validate()
}

func (h *Handler) loadPipelineForDAG(w http.ResponseWriter, r *http.Request) (*Pipeline, bool) {
	id := r.PathValue("id")
	if strings.TrimSpace(id) == "" {
		http.Error(w, "Pipeline ID is required", http.StatusBadRequest)
		return nil, false
	}

	p, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			http.Error(w, "Pipeline not found", http.StatusNotFound)
			return nil, false
		}
		http.Error(w, "Failed to get pipeline", http.StatusInternalServerError)
		return nil, false
	}
	return p, true
}

func renderDAGWorkspace(w http.ResponseWriter, r *http.Request, p *Pipeline, formErr string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = components.DAGWorkspace(buildDAGView(p), formErr).Render(r.Context(), w)
}

func (h *Handler) GetDAG(w http.ResponseWriter, r *http.Request) {
	p, ok := h.loadPipelineForDAG(w, r)
	if !ok {
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := pages.PipelineDAG(buildDAGView(p)).Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render DAG page", http.StatusInternalServerError)
	}
}

func extractNodeColumns(n Node) []string {
	switch n.Type {
	case "db_source":
		var cfg stage.DBSourceConfig
		if json.Unmarshal(n.Config, &cfg) == nil {
			return cfg.Columns
		}
	case "db_sink":
		var cfg stage.DBSinkConfig
		if json.Unmarshal(n.Config, &cfg) == nil {
			return cfg.Columns
		}
	case "transform":
		var cfg stage.TransformConfig
		if json.Unmarshal(n.Config, &cfg) == nil {
			cols := make([]string, 0, len(cfg.Rename)+len(cfg.Set))
			renameKeys := make([]string, 0, len(cfg.Rename))
			for k := range cfg.Rename {
				renameKeys = append(renameKeys, k)
			}
			slices.Sort(renameKeys)
			for _, k := range renameKeys {
				cols = append(cols, fmt.Sprintf("%s -> %s", k, cfg.Rename[k]))
			}
			setKeys := make([]string, 0, len(cfg.Set))
			for k := range cfg.Set {
				setKeys = append(setKeys, k)
			}
			slices.Sort(setKeys)
			for _, k := range setKeys {
				cols = append(cols, "+"+k)
			}
			return cols
		}
	}
	return nil
}

func collectTransformedColumns(p *Pipeline) []string {
	cols := make([]string, 0)
	addUnique := func(c string) {
		if c != "" && !slices.Contains(cols, c) {
			cols = append(cols, c)
		}
	}

	for _, n := range p.Nodes {
		if n.Type == "db_source" {
			var cfg stage.DBSourceConfig
			if json.Unmarshal(n.Config, &cfg) == nil {
				for _, c := range cfg.Columns {
					addUnique(c)
				}
			}
		}
	}

	for _, n := range p.Nodes {
		if n.Type == "transform" {
			var cfg stage.TransformConfig
			if json.Unmarshal(n.Config, &cfg) == nil {
				for i, c := range cols {
					if newName, ok := cfg.Rename[c]; ok && newName != "" {
						cols[i] = newName
					}
				}
				setKeys := make([]string, 0, len(cfg.Set))
				for k := range cfg.Set {
					setKeys = append(setKeys, k)
				}
				slices.Sort(setKeys)
				for _, k := range setKeys {
					addUnique(k)
				}
			}
		}
	}

	return cols
}

func parseCommaList(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func buildStageConfigFromForm(nodeType string, r *http.Request) (json.RawMessage, error) {
	if rawConfig := strings.TrimSpace(r.FormValue("config")); rawConfig != "" {
		if !json.Valid([]byte(rawConfig)) {
			return nil, errors.New("Stage config must be valid JSON")
		}
		return json.RawMessage(rawConfig), nil
	}

	switch nodeType {
	case "db_source":
		cols := parseCommaList(r.FormValue("columns"))
		query := strings.TrimSpace(r.FormValue("query"))
		driver := strings.TrimSpace(r.FormValue("driver"))
		dsn := strings.TrimSpace(r.FormValue("dsn"))
		if len(cols) == 0 && query != "" {
			cols = stage.ParseSelectColumns(query)
			if len(cols) == 0 && dsn != "" {
				if introspected, err := stage.IntrospectQueryColumns(r.Context(), driver, dsn, query); err == nil {
					cols = introspected
				}
			}
		}
		cfg := stage.DBSourceConfig{
			Driver:  driver,
			DSN:     dsn,
			Query:   query,
			Columns: cols,
		}
		return json.Marshal(cfg)
	case "transform":
		rename := make(map[string]string)
		renameFrom := r.Form["rename_from[]"]
		renameTo := r.Form["rename_to[]"]
		for i := 0; i < len(renameFrom) && i < len(renameTo); i++ {
			from := strings.TrimSpace(renameFrom[i])
			to := strings.TrimSpace(renameTo[i])
			if from != "" && to != "" && from != to {
				rename[from] = to
			}
		}
		if len(renameFrom) == 0 {
			if raw := strings.TrimSpace(r.FormValue("rename")); raw != "" {
				if err := json.Unmarshal([]byte(raw), &rename); err != nil {
					return nil, errors.New("Rename must be a valid JSON object")
				}
			}
		}

		set := make(map[string]any)
		setKeys := r.Form["set_key[]"]
		setVals := r.Form["set_val[]"]
		for i := 0; i < len(setKeys) && i < len(setVals); i++ {
			k := strings.TrimSpace(setKeys[i])
			v := strings.TrimSpace(setVals[i])
			if k != "" {
				set[k] = v
			}
		}
		if len(setKeys) == 0 {
			if raw := strings.TrimSpace(r.FormValue("set")); raw != "" {
				if err := json.Unmarshal([]byte(raw), &set); err != nil {
					return nil, errors.New("Set must be a valid JSON object")
				}
			}
		}
		return json.Marshal(stage.TransformConfig{Rename: rename, Set: set})
	case "db_sink":
		cfg := stage.DBSinkConfig{
			Driver:  strings.TrimSpace(r.FormValue("driver")),
			DSN:     strings.TrimSpace(r.FormValue("dsn")),
			Table:   strings.TrimSpace(r.FormValue("table")),
			Columns: parseCommaList(r.FormValue("columns")),
		}
		return json.Marshal(cfg)
	default:
		return json.RawMessage("{}"), nil
	}
}

func (h *Handler) GetSQLColumns(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("query"))
	driver := strings.TrimSpace(r.URL.Query().Get("driver"))
	dsn := strings.TrimSpace(r.URL.Query().Get("dsn"))

	cols := stage.ParseSelectColumns(query)
	if len(cols) == 0 && query != "" && dsn != "" {
		if introspected, err := stage.IntrospectQueryColumns(r.Context(), driver, dsn, query); err == nil {
			cols = introspected
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = components.OutputColumnsInput(strings.Join(cols, ", ")).Render(r.Context(), w)
}

func (h *Handler) GetStageConfigFields(w http.ResponseWriter, r *http.Request) {
	p, ok := h.loadPipelineForDAG(w, r)
	if !ok {
		return
	}

	stageType := strings.TrimSpace(r.URL.Query().Get("type"))
	transformedCols := collectTransformedColumns(p)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = components.StageConfigFields(stageType, transformedCols).Render(r.Context(), w)
}

func (h *Handler) AddNode(w http.ResponseWriter, r *http.Request) {
	p, ok := h.loadPipelineForDAG(w, r)
	if !ok {
		return
	}

	if err := r.ParseForm(); err != nil {
		renderDAGWorkspace(w, r, p, "Invalid form data")
		return
	}

	nodeID := strings.TrimSpace(r.FormValue("id"))
	nodeType := strings.TrimSpace(r.FormValue("type"))

	if nodeID == "" {
		renderDAGWorkspace(w, r, p, "Stage ID is required")
		return
	}
	if nodeType == "" {
		renderDAGWorkspace(w, r, p, "Stage type is required")
		return
	}

	cfgBytes, err := buildStageConfigFromForm(nodeType, r)
	if err != nil {
		renderDAGWorkspace(w, r, p, err.Error())
		return
	}

	if _, err := stage.NewDefaultRegistry().Create(nodeType, cfgBytes); err != nil {
		renderDAGWorkspace(w, r, p, err.Error())
		return
	}

	for _, existing := range p.Nodes {
		if existing.ID == nodeID {
			renderDAGWorkspace(w, r, p, "Stage ID already exists: "+nodeID)
			return
		}
	}

	p.Nodes = append(p.Nodes, Node{
		ID:     nodeID,
		Type:   nodeType,
		Config: cfgBytes,
	})

	if err := h.repo.Update(r.Context(), p); err != nil {
		http.Error(w, "Failed to update pipeline", http.StatusInternalServerError)
		return
	}

	renderDAGWorkspace(w, r, p, "")
}

func extractStageConfigValues(n Node) components.StageConfigValues {
	var vals components.StageConfigValues
	switch n.Type {
	case "db_source":
		var cfg stage.DBSourceConfig
		if json.Unmarshal(n.Config, &cfg) == nil {
			vals.Driver = cfg.Driver
			vals.DSN = cfg.DSN
			vals.Query = cfg.Query
			vals.Columns = strings.Join(cfg.Columns, ", ")
		}
	case "transform":
		var cfg stage.TransformConfig
		if json.Unmarshal(n.Config, &cfg) == nil {
			renameKeys := make([]string, 0, len(cfg.Rename))
			for k := range cfg.Rename {
				renameKeys = append(renameKeys, k)
			}
			slices.Sort(renameKeys)
			for _, k := range renameKeys {
				vals.RenamePairs = append(vals.RenamePairs, components.MappingPair{Key: k, Value: cfg.Rename[k]})
			}

			setKeys := make([]string, 0, len(cfg.Set))
			for k := range cfg.Set {
				setKeys = append(setKeys, k)
			}
			slices.Sort(setKeys)
			for _, k := range setKeys {
				vals.SetPairs = append(vals.SetPairs, components.MappingPair{Key: k, Value: fmt.Sprint(cfg.Set[k])})
			}
		}
	case "db_sink":
		var cfg stage.DBSinkConfig
		if json.Unmarshal(n.Config, &cfg) == nil {
			vals.Driver = cfg.Driver
			vals.DSN = cfg.DSN
			vals.Table = cfg.Table
			vals.Columns = strings.Join(cfg.Columns, ", ")
		}
	}
	return vals
}

func (h *Handler) GetNode(w http.ResponseWriter, r *http.Request) {
	p, ok := h.loadPipelineForDAG(w, r)
	if !ok {
		return
	}

	nodeID := r.PathValue("nodeId")
	for _, n := range p.Nodes {
		if n.ID == nodeID {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_ = components.NodeEditCard(
				p.ID,
				n.ID,
				n.Type,
				extractStageConfigValues(n),
				collectTransformedColumns(p),
				"",
			).Render(r.Context(), w)
			return
		}
	}

	http.Error(w, "Stage not found", http.StatusNotFound)
}

func (h *Handler) UpdateNode(w http.ResponseWriter, r *http.Request) {
	p, ok := h.loadPipelineForDAG(w, r)
	if !ok {
		return
	}

	nodeID := r.PathValue("nodeId")
	idx := -1
	for i, n := range p.Nodes {
		if n.ID == nodeID {
			idx = i
			break
		}
	}
	if idx == -1 {
		http.Error(w, "Stage not found", http.StatusNotFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		renderDAGWorkspace(w, r, p, "Invalid form data")
		return
	}

	nodeType := strings.TrimSpace(r.FormValue("type"))
	if nodeType == "" {
		renderDAGWorkspace(w, r, p, "Stage type is required")
		return
	}

	cfgBytes, err := buildStageConfigFromForm(nodeType, r)
	if err != nil {
		renderDAGWorkspace(w, r, p, err.Error())
		return
	}

	if _, err := stage.NewDefaultRegistry().Create(nodeType, cfgBytes); err != nil {
		renderDAGWorkspace(w, r, p, err.Error())
		return
	}

	p.Nodes[idx].Type = nodeType
	p.Nodes[idx].Config = cfgBytes

	if err := h.repo.Update(r.Context(), p); err != nil {
		http.Error(w, "Failed to update pipeline", http.StatusInternalServerError)
		return
	}

	renderDAGWorkspace(w, r, p, "")
}

func (h *Handler) DeleteNode(w http.ResponseWriter, r *http.Request) {
	p, ok := h.loadPipelineForDAG(w, r)
	if !ok {
		return
	}

	nodeID := r.PathValue("nodeId")
	if strings.TrimSpace(nodeID) == "" {
		renderDAGWorkspace(w, r, p, "Stage ID is required")
		return
	}

	p.Nodes = slices.DeleteFunc(p.Nodes, func(n Node) bool {
		return n.ID == nodeID
	})
	p.Edges = slices.DeleteFunc(p.Edges, func(e Edge) bool {
		return e.From == nodeID || e.To == nodeID
	})

	if err := h.repo.Update(r.Context(), p); err != nil {
		http.Error(w, "Failed to update pipeline", http.StatusInternalServerError)
		return
	}

	renderDAGWorkspace(w, r, p, "")
}

func (h *Handler) AddEdge(w http.ResponseWriter, r *http.Request) {
	p, ok := h.loadPipelineForDAG(w, r)
	if !ok {
		return
	}

	if err := r.ParseForm(); err != nil {
		renderDAGWorkspace(w, r, p, "Invalid form data")
		return
	}

	from := strings.TrimSpace(r.FormValue("from"))
	to := strings.TrimSpace(r.FormValue("to"))

	candidate := *p
	candidate.Edges = append(slices.Clone(p.Edges), Edge{From: from, To: to})

	if err := validatePipelineDAG(&candidate); err != nil {
		renderDAGWorkspace(w, r, p, err.Error())
		return
	}

	if err := h.repo.Update(r.Context(), &candidate); err != nil {
		http.Error(w, "Failed to update pipeline", http.StatusInternalServerError)
		return
	}

	renderDAGWorkspace(w, r, &candidate, "")
}

func (h *Handler) DeleteEdge(w http.ResponseWriter, r *http.Request) {
	p, ok := h.loadPipelineForDAG(w, r)
	if !ok {
		return
	}

	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")

	p.Edges = slices.DeleteFunc(p.Edges, func(e Edge) bool {
		return e.From == from && e.To == to
	})

	if err := h.repo.Update(r.Context(), p); err != nil {
		http.Error(w, "Failed to update pipeline", http.StatusInternalServerError)
		return
	}

	renderDAGWorkspace(w, r, p, "")
}
