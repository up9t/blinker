package backend

type BreakpointService struct{}

type DurationUnitType = int

const (
	DurationUnitSecond DurationUnitType = iota
	DurationUnitMinute
)

type Breakpoint struct {
	id           string
	interval     int
	intervalUnit DurationUnitType
	duration     int
	durationUnit DurationUnitType
	enabled      bool
}

func NewBreakpointService() *BreakpointService {
	return &BreakpointService{}
}

func (svc *BreakpointService) Add(bp Breakpoint) {

}

// func (svc *BreakpointService) GetByID(id string) Breakpoint {

// }

// func (svc *BreakpointService) List() []Breakpoint {

// }

func (svc *BreakpointService) Disable() {

}

func (svc *BreakpointService) Enable() {

}

func (svc *BreakpointService) Remove() {

}

func (svc *BreakpointService) Update() {

}
