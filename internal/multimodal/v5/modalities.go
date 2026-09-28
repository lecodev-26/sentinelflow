package multimodalv5

type Modality string

const (
	Text     Modality = "text"
	Image    Modality = "image"
	Audio    Modality = "audio"
	Video    Modality = "video"
	Document Modality = "document"
)

type Part struct {
	Modality Modality
	URI      string
	MimeType string
	Data     []byte
}
type Request struct{ Parts []Part }

func (r Request) Has(m Modality) bool {
	for _, p := range r.Parts {
		if p.Modality == m {
			return true
		}
	}
	return false
}
func (r Request) Validate() error {
	for _, p := range r.Parts {
		if p.URI == "" && len(p.Data) == 0 {
			return &ValidationError{"empty part"}
		}
	}
	return nil
}

type ValidationError struct{ Reason string }

func (e *ValidationError) Error() string { return e.Reason }
