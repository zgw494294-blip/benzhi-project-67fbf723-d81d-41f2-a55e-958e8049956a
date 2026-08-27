package web

import (
	"embed"
	"net/http"
	"seed-germination-workbench/internal/domain"
	"seed-germination-workbench/internal/workflow"
	"strings"
	"time"
)

//go:embed static/*
var assets embed.FS

type Server struct {
	svc  *workflow.Service
	http *http.Server
}

func New(svc *workflow.Service, addr string) *Server {
	mux := http.NewServeMux()
	w := &Server{svc: svc}
	mux.HandleFunc("/", w.index)
	mux.Handle("/static/", http.FileServer(http.FS(assets)))
	mux.HandleFunc("/api/trials", w.trials)
	mux.HandleFunc("/api/trials/", w.trial)
	w.http = &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	return w
}
func (s *Server) ListenAndServe() error { return s.http.ListenAndServe() }
func (s *Server) Shutdown() error       { return s.http.Close() }
func (s *Server) index(rw http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(rw, r)
		return
	}
	b, _ := assets.ReadFile("static/index.html")
	rw.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = rw.Write(b)
}

func (s *Server) trials(rw http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		result, err := s.svc.Search(workflow.ListFilter{Species: r.URL.Query().Get("species"), CollectionBatch: r.URL.Query().Get("collectionBatch"), Status: domain.Status(r.URL.Query().Get("status"))})
		if err != nil {
			errJSON(rw, err)
			return
		}
		writeJSON(rw, http.StatusOK, result)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(rw, "method", http.StatusMethodNotAllowed)
		return
	}
	var in struct {
		TrialID           string `json:"trialId"`
		SpeciesName       string `json:"speciesName"`
		AccessionCode     string `json:"accessionCode"`
		CollectionBatch   string `json:"collectionBatch"`
		ReplicateCount    int    `json:"replicateCount"`
		SeedsPerReplicate int    `json:"seedsPerReplicate"`
		Actor             string `json:"actor"`
	}
	if decode(r, inPtr(&in)) != nil {
		errJSON(rw, ErrBad)
		return
	}
	t, err := s.svc.CreateBy(domain.Trial{TrialID: in.TrialID, SpeciesName: in.SpeciesName, AccessionCode: in.AccessionCode, CollectionBatch: in.CollectionBatch, ReplicateCount: in.ReplicateCount, SeedsPerReplicate: in.SeedsPerReplicate}, in.Actor)
	if err != nil {
		errJSON(rw, err)
		return
	}
	writeJSON(rw, http.StatusCreated, t)
}

var ErrBad = &apiError{"请求格式无效"}

type apiError struct{ msg string }

func (e *apiError) Error() string { return e.msg }

type actionInput struct {
	ExpectedRevision            int64                     `json:"expectedRevision"`
	Actor                       string                    `json:"actor"`
	ProtocolID                  string                    `json:"protocolId"`
	StratificationDays          int                       `json:"stratificationDays"`
	TemperatureCelsius          float64                   `json:"temperatureCelsius"`
	LightRegime                 string                    `json:"lightRegime"`
	Substrate                   string                    `json:"substrate"`
	ObservationDays             int                       `json:"observationDays"`
	GerminationThresholdPercent float64                   `json:"germinationThresholdPercent"`
	ContentDigest               string                    `json:"contentDigest"`
	DayIndex                    int                       `json:"dayIndex"`
	Observations                []domain.DailyObservation `json:"observations"`
	Kind                        string                    `json:"kind"`
	Description                 string                    `json:"description"`
	CorrectiveAction            string                    `json:"correctiveAction"`
	WindowStart                 int                       `json:"windowStart"`
	WindowEnd                   int                       `json:"windowEnd"`
	DeviationID                 string                    `json:"deviationId"`
	ResponsiblePerson           string                    `json:"responsiblePerson"`
	CompletionDescription       string                    `json:"completionDescription"`
	ObservationIDs              []string                  `json:"observationIds"`
	Decision                    string                    `json:"decision"`
	Conclusion                  string                    `json:"conclusion"`
	Reason                      string                    `json:"reason"`
	Reviewer                    string                    `json:"reviewer"`
	SnapshotDigest              string                    `json:"snapshotDigest"`
	Issues                      []struct {
		Category       string `json:"category"`
		Description    string `json:"description"`
		RequiredAction string `json:"requiredAction"`
		ObjectType     string `json:"objectType"`
		ObjectID       string `json:"objectId"`
	} `json:"issues"`
	IssueID        string   `json:"issueId"`
	CorrectionNote string   `json:"correctionNote"`
	ReferenceIDs   []string `json:"referenceIds"`
	Confirm        bool     `json:"confirm"`
}

func (s *Server) trial(rw http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 3 {
		http.NotFound(rw, r)
		return
	}
	id, action := parts[2], ""
	if len(parts) > 3 {
		action = parts[3]
	}
	if r.Method == http.MethodGet {
		s.getTrial(rw, r, id, action)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(rw, "method", http.StatusMethodNotAllowed)
		return
	}
	var in actionInput
	if decode(r, &in) != nil {
		errJSON(rw, ErrBad)
		return
	}
	if action == "protocol-preview" {
		preview, err := s.svc.PreviewProtocol(id, in.protocol())
		if err != nil {
			errJSON(rw, err)
			return
		}
		writeJSON(rw, http.StatusOK, preview)
		return
	}
	key := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if key == "" {
		errJSON(rw, &apiError{"缺少 Idempotency-Key"})
		return
	}
	if in.ExpectedRevision < 1 {
		errJSON(rw, &apiError{"expectedRevision 必须为当前版本"})
		return
	}
	actor := strings.TrimSpace(in.Actor)
	if actor == "" {
		actor = "system"
	}
	var out *domain.Trial
	var err error
	switch action {
	case "protocol":
		out, err = s.svc.LockProtocolChecked(id, key, in.ExpectedRevision, in.ContentDigest, actor, in.protocol())
	case "start":
		out, err = s.svc.StartChecked(id, key, in.ExpectedRevision, actor)
	case "observe":
		out, err = s.svc.ObserveBatch(id, key, in.ExpectedRevision, in.DayIndex, in.Observations, actor)
	case "deviations":
		out, err = s.svc.DeviationChecked(id, key, in.ExpectedRevision, domain.Deviation{Kind: in.Kind, Description: in.Description, CorrectiveAction: in.CorrectiveAction, WindowStart: in.WindowStart, WindowEnd: in.WindowEnd}, actor)
	case "resolve":
		out, err = s.svc.ResolveChecked(id, key, in.ExpectedRevision, in.DeviationID, in.ResponsiblePerson, in.CompletionDescription, in.ObservationIDs, actor)
	case "submit":
		out, err = s.svc.SubmitChecked(id, key, in.ExpectedRevision, actor)
	case "review":
		issues := make([]domain.ReviewIssueInput, 0, len(in.Issues))
		for _, x := range in.Issues {
			issues = append(issues, domain.ReviewIssueInput{Category: x.Category, Description: x.Description, RequiredAction: x.RequiredAction, ObjectType: x.ObjectType, ObjectID: x.ObjectID})
		}
		conclusion := in.Conclusion
		if conclusion == "" {
			conclusion = in.Reason
		}
		out, err = s.svc.ReviewChecked(id, key, in.ExpectedRevision, in.Decision, conclusion, in.Reviewer, in.SnapshotDigest, issues)
	case "corrections":
		out, err = s.svc.Correct(id, key, in.ExpectedRevision, in.IssueID, in.CorrectionNote, in.ReferenceIDs, in.Confirm, actor)
	default:
		http.NotFound(rw, r)
		return
	}
	if err != nil {
		errJSON(rw, err)
		return
	}
	writeJSON(rw, http.StatusOK, out)
}

func (s *Server) getTrial(rw http.ResponseWriter, r *http.Request, id, action string) {
	if action == "details" {
		f := workflow.EventFilter{Type: r.URL.Query().Get("eventType"), Actor: r.URL.Query().Get("actor")}
		if v := r.URL.Query().Get("from"); v != "" {
			if x, e := time.Parse(time.RFC3339, v); e == nil {
				f.From = &x
			}
		}
		if v := r.URL.Query().Get("to"); v != "" {
			if x, e := time.Parse(time.RFC3339, v); e == nil {
				f.To = &x
			}
		}
		result, err := s.svc.Details(id, f)
		if err != nil {
			errJSON(rw, err)
			return
		}
		writeJSON(rw, http.StatusOK, result)
		return
	}
	if action != "" {
		http.NotFound(rw, r)
		return
	}
	t, err := s.svc.Get(id)
	if err != nil {
		errJSON(rw, err)
		return
	}
	writeJSON(rw, http.StatusOK, t)
}
