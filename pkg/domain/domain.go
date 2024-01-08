package domain

type Profile struct {
	Languages []Language `json:"languages"`
}

type Language struct {
	ID         string      `json:"id"`
	ShowName   string      `json:"show_name"`
	Processors []Processor `json:"processors"`
}

type Processor struct {
	ID       string `json:"id"`
	ShowName string `json:"show_name"`

	DockerImage string `json:"docker_image"`

	DefaultFilename string `json:"default_filename"`

	Tasks []Task `json:"tasks"`
}

type Task struct {
	ID       string `json:"id"`
	ShowName string `json:"show_name"`

	Kind string `json:"kind"` // "action" | "tool"

	Compile *PhasedTask `json:"compile,omitempty"`
	Run     *PhasedTask `json:"run,omitempty"`
}

type PhasedTask struct {
	Cmd []string `json:"cmd"`
}
