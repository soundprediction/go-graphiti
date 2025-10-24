package prompts

// Library defines the interface for the complete prompt library.
type Library interface {
	ExtractNodes() ExtractNodesPrompt
	DedupeNodes() DedupeNodesPrompt
	ExtractEdges() ExtractEdgesPrompt
	DedupeEdges() DedupeEdgesPrompt
	InvalidateEdges() InvalidateEdgesPrompt
	ExtractEdgeDates() ExtractEdgeDatesPrompt
	SummarizeNodes() SummarizeNodesPrompt
	Eval() EvalPrompt
}

// LibraryImpl implements the Library interface.
type LibraryImpl struct {
	extractNodes     ExtractNodesPrompt
	dedupeNodes      DedupeNodesPrompt
	extractEdges     ExtractEdgesPrompt
	dedupeEdges      DedupeEdgesPrompt
	invalidateEdges  InvalidateEdgesPrompt
	extractEdgeDates ExtractEdgeDatesPrompt
	summarizeNodes   SummarizeNodesPrompt
	eval             EvalPrompt
}

func (l *LibraryImpl) ExtractNodes() ExtractNodesPrompt         { return l.extractNodes }
func (l *LibraryImpl) DedupeNodes() DedupeNodesPrompt           { return l.dedupeNodes }
func (l *LibraryImpl) ExtractEdges() ExtractEdgesPrompt         { return l.extractEdges }
func (l *LibraryImpl) DedupeEdges() DedupeEdgesPrompt           { return l.dedupeEdges }
func (l *LibraryImpl) InvalidateEdges() InvalidateEdgesPrompt   { return l.invalidateEdges }
func (l *LibraryImpl) ExtractEdgeDates() ExtractEdgeDatesPrompt { return l.extractEdgeDates }
func (l *LibraryImpl) SummarizeNodes() SummarizeNodesPrompt     { return l.summarizeNodes }
func (l *LibraryImpl) Eval() EvalPrompt                         { return l.eval }

// NewLibrary creates a new prompt library instance.
func NewLibrary() Library {
	return &LibraryImpl{
		extractNodes:     NewExtractNodesVersions(),
		dedupeNodes:      NewDedupeNodesVersions(),
		extractEdges:     NewExtractEdgesVersions(),
		dedupeEdges:      NewDedupeEdgesVersions(),
		invalidateEdges:  NewInvalidateEdgesVersions(),
		extractEdgeDates: NewExtractEdgeDatesVersions(),
		summarizeNodes:   NewSummarizeNodesVersions(),
		eval:             NewEvalVersions(),
	}
}

// DefaultLibrary is the default prompt library instance.
var DefaultLibrary = NewLibrary()
