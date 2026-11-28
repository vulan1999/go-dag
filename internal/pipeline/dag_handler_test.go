package pipeline

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func setupDAGTestHandler(t *testing.T) (*Handler, *Pipeline, func()) {
	t.Helper()
	repo, cleanup := setupTestDB(t)
	h := NewHandler(repo)

	p := &Pipeline{
		ID:    "pipe-1",
		Name:  "ETL Pipeline",
		Nodes: []Node{},
		Edges: []Edge{},
	}
	if err := repo.Create(context.Background(), p); err != nil {
		cleanup()
		t.Fatalf("failed to create test pipeline: %v", err)
	}
	return h, p, cleanup
}

func TestHandler_GetDAG(t *testing.T) {
	h, _, cleanup := setupDAGTestHandler(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/pipelines/pipe-1/dag", nil)
	req.SetPathValue("id", "pipe-1")
	rec := httptest.NewRecorder()

	h.GetDAG(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "ETL Pipeline") || !strings.Contains(body, `id="dag-workspace"`) {
		t.Errorf("expected DAG page to contain pipeline name and workspace, got: %s", body)
	}
}

func TestHandler_GetDAG_NotFound(t *testing.T) {
	h, _, cleanup := setupDAGTestHandler(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/pipelines/missing/dag", nil)
	req.SetPathValue("id", "missing")
	rec := httptest.NewRecorder()

	h.GetDAG(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestHandler_AddAndDeleteNode(t *testing.T) {
	h, _, cleanup := setupDAGTestHandler(t)
	defer cleanup()

	// Add node
	form := url.Values{}
	form.Set("id", "extract")
	form.Set("type", "source")
	form.Set("config", `{"table":"users"}`)

	req := httptest.NewRequest(http.MethodPost, "/pipelines/pipe-1/nodes", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("id", "pipe-1")
	rec := httptest.NewRecorder()

	h.AddNode(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "extract") {
		t.Fatalf("expected workspace to render added node")
	}

	// Duplicate node should render error message without saving
	reqDup := httptest.NewRequest(http.MethodPost, "/pipelines/pipe-1/nodes", strings.NewReader(form.Encode()))
	reqDup.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqDup.SetPathValue("id", "pipe-1")
	recDup := httptest.NewRecorder()

	h.AddNode(recDup, reqDup)
	if !strings.Contains(recDup.Body.String(), "Stage ID already exists") {
		t.Errorf("expected duplicate error message, got: %s", recDup.Body.String())
	}

	// Delete node
	reqDel := httptest.NewRequest(http.MethodDelete, "/pipelines/pipe-1/nodes/extract", nil)
	reqDel.SetPathValue("id", "pipe-1")
	reqDel.SetPathValue("nodeId", "extract")
	recDel := httptest.NewRecorder()

	h.DeleteNode(recDel, reqDel)
	if recDel.Code != http.StatusOK {
		t.Fatalf("expected 200 on delete, got %d", recDel.Code)
	}
	if strings.Contains(recDel.Body.String(), `data-node-id="extract"`) {
		t.Errorf("expected node to be removed from workspace")
	}
}

func TestHandler_AddAndDeleteEdge_CycleValidation(t *testing.T) {
	h, _, cleanup := setupDAGTestHandler(t)
	defer cleanup()

	// Seed two nodes
	for _, n := range []struct{ id, typ string }{{"a", "source"}, {"b", "sink"}} {
		f := url.Values{"id": {n.id}, "type": {n.typ}, "config": {"{}"}}
		r := httptest.NewRequest(http.MethodPost, "/pipelines/pipe-1/nodes", strings.NewReader(f.Encode()))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		r.SetPathValue("id", "pipe-1")
		h.AddNode(httptest.NewRecorder(), r)
	}

	// Add valid edge a -> b
	edgeForm := url.Values{"from": {"a"}, "to": {"b"}}
	req := httptest.NewRequest(http.MethodPost, "/pipelines/pipe-1/edges", strings.NewReader(edgeForm.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetPathValue("id", "pipe-1")
	rec := httptest.NewRecorder()
	h.AddEdge(rec, req)

	if !strings.Contains(rec.Body.String(), "Valid DAG") {
		t.Fatalf("expected Valid DAG after adding a -> b")
	}

	// Attempt cycle b -> a
	cycleForm := url.Values{"from": {"b"}, "to": {"a"}}
	reqCycle := httptest.NewRequest(http.MethodPost, "/pipelines/pipe-1/edges", strings.NewReader(cycleForm.Encode()))
	reqCycle.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqCycle.SetPathValue("id", "pipe-1")
	recCycle := httptest.NewRecorder()
	h.AddEdge(recCycle, reqCycle)

	if !strings.Contains(recCycle.Body.String(), "cycle detected in DAG") {
		t.Errorf("expected cycle error message, got: %s", recCycle.Body.String())
	}

	// Delete edge a -> b
	reqDel := httptest.NewRequest(http.MethodDelete, "/pipelines/pipe-1/edges?from=a&to=b", nil)
	reqDel.SetPathValue("id", "pipe-1")
	recDel := httptest.NewRecorder()
	h.DeleteEdge(recDel, reqDel)

	if recDel.Code != http.StatusOK {
		t.Fatalf("expected 200 on delete edge, got %d", recDel.Code)
	}
	if strings.Contains(recDel.Body.String(), `class="edge-item"`) {
		t.Errorf("expected edge list to be empty after removal")
	}
}

func TestHandler_StageConfigFieldsAndTransformedColumns(t *testing.T) {
	h, _, cleanup := setupDAGTestHandler(t)
	defer cleanup()

	// 1. Add db_source using structured form fields
	srcForm := url.Values{
		"id":      {"src_a"},
		"type":    {"db_source"},
		"driver":  {"sqlite3"},
		"dsn":     {"db_a.sqlite"},
		"query":   {"SELECT id, name FROM users"},
		"columns": {"id, name"},
	}
	r1 := httptest.NewRequest(http.MethodPost, "/pipelines/pipe-1/nodes", strings.NewReader(srcForm.Encode()))
	r1.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r1.SetPathValue("id", "pipe-1")
	rec1 := httptest.NewRecorder()
	h.AddNode(rec1, r1)

	if !strings.Contains(rec1.Body.String(), "col-chip") || !strings.Contains(rec1.Body.String(), "name") {
		t.Fatalf("expected db_source columns to render on node card, got: %s", rec1.Body.String())
	}

	// 2. Add transform renaming name -> full_name
	trForm := url.Values{
		"id":     {"tr_1"},
		"type":   {"transform"},
		"rename": {`{"name":"full_name"}`},
		"set":    {`{}`},
	}
	r2 := httptest.NewRequest(http.MethodPost, "/pipelines/pipe-1/nodes", strings.NewReader(trForm.Encode()))
	r2.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r2.SetPathValue("id", "pipe-1")
	rec2 := httptest.NewRecorder()
	h.AddNode(rec2, r2)

	if !strings.Contains(rec2.Body.String(), "name -&gt; full_name") && !strings.Contains(rec2.Body.String(), "name -> full_name") {
		t.Fatalf("expected transform rename chip on node card, got: %s", rec2.Body.String())
	}

	// 3. Request GetStageConfigFields for db_sink and verify transformed columns are shown
	r3 := httptest.NewRequest(http.MethodGet, "/pipelines/pipe-1/stages/config-fields?type=db_sink", nil)
	r3.SetPathValue("id", "pipe-1")
	rec3 := httptest.NewRecorder()
	h.GetStageConfigFields(rec3, r3)

	if rec3.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec3.Code)
	}
	if !strings.Contains(rec3.Body.String(), "id, full_name") {
		t.Errorf("expected db_sink config fields to show transformed columns 'id, full_name', got: %s", rec3.Body.String())
	}

	// 4. GetNode returns pre-filled edit card
	rGet := httptest.NewRequest(http.MethodGet, "/pipelines/pipe-1/nodes/src_a", nil)
	rGet.SetPathValue("id", "pipe-1")
	rGet.SetPathValue("nodeId", "src_a")
	recGet := httptest.NewRecorder()
	h.GetNode(recGet, rGet)

	if recGet.Code != http.StatusOK {
		t.Fatalf("expected 200 from GetNode, got %d", recGet.Code)
	}
	if !strings.Contains(recGet.Body.String(), "db_a.sqlite") || !strings.Contains(recGet.Body.String(), `hx-put="/pipelines/pipe-1/nodes/src_a"`) {
		t.Errorf("expected GetNode to render pre-filled edit card, got: %s", recGet.Body.String())
	}

	// 5. UpdateNode updates node properties and refreshes workspace
	updForm := url.Values{
		"type":    {"db_source"},
		"driver":  {"sqlite3"},
		"dsn":     {"db_a_v2.sqlite"},
		"query":   {"SELECT id, name, email FROM users"},
		"columns": {"id, name, email"},
	}
	rUpd := httptest.NewRequest(http.MethodPut, "/pipelines/pipe-1/nodes/src_a", strings.NewReader(updForm.Encode()))
	rUpd.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rUpd.SetPathValue("id", "pipe-1")
	rUpd.SetPathValue("nodeId", "src_a")
	recUpd := httptest.NewRecorder()
	h.UpdateNode(recUpd, rUpd)

	if recUpd.Code != http.StatusOK {
		t.Fatalf("expected 200 from UpdateNode, got %d", recUpd.Code)
	}
	if !strings.Contains(recUpd.Body.String(), "db_a_v2.sqlite") || !strings.Contains(recUpd.Body.String(), "email") {
		t.Errorf("expected UpdateNode to render updated node properties, got: %s", recUpd.Body.String())
	}
}

func TestHandler_GetSQLColumns_And_AutoFillOnAdd(t *testing.T) {
	h, _, cleanup := setupDAGTestHandler(t)
	defer cleanup()

	// Test GetSQLColumns with nested query
	q := url.QueryEscape("SELECT id, total_amount FROM (SELECT user_id AS id, SUM(amount) AS total_amount FROM orders GROUP BY user_id) sub")
	req := httptest.NewRequest(http.MethodGet, "/stages/sql-columns?query="+q, nil)
	rec := httptest.NewRecorder()
	h.GetSQLColumns(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `value="id, total_amount"`) {
		t.Errorf("expected auto-filled columns input, got: %s", rec.Body.String())
	}

	// Test AddNode with db_source and empty columns auto-fills from SQL
	srcForm := url.Values{
		"id":     {"src_auto"},
		"type":   {"db_source"},
		"driver": {"sqlite3"},
		"dsn":    {"db_a.sqlite"},
		"query":  {"WITH cte AS (SELECT 1) SELECT u.id, u.email AS contact_email FROM users u"},
	}
	rAdd := httptest.NewRequest(http.MethodPost, "/pipelines/pipe-1/nodes", strings.NewReader(srcForm.Encode()))
	rAdd.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rAdd.SetPathValue("id", "pipe-1")
	recAdd := httptest.NewRecorder()
	h.AddNode(recAdd, rAdd)

	if !strings.Contains(recAdd.Body.String(), "contact_email") {
		t.Errorf("expected auto-filled contact_email chip on node card, got: %s", recAdd.Body.String())
	}

	// Test AddNode with transform column mapping arrays
	trForm := url.Values{
		"id":            {"tr_map"},
		"type":          {"transform"},
		"rename_from[]": {"id", "contact_email"},
		"rename_to[]":   {"id", "email"},
		"set_key[]":     {"env"},
		"set_val[]":     {"prod"},
	}
	rTr := httptest.NewRequest(http.MethodPost, "/pipelines/pipe-1/nodes", strings.NewReader(trForm.Encode()))
	rTr.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rTr.SetPathValue("id", "pipe-1")
	recTr := httptest.NewRecorder()
	h.AddNode(recTr, rTr)

	bodyTr := recTr.Body.String()
	if !strings.Contains(bodyTr, "contact_email -&gt; email") && !strings.Contains(bodyTr, "contact_email -> email") {
		t.Errorf("expected rename chip contact_email -> email, got: %s", bodyTr)
	}
	if !strings.Contains(bodyTr, "+env") {
		t.Errorf("expected static field chip +env, got: %s", bodyTr)
	}
}
