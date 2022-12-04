package routes

type GenericResponse struct {
	Message string `json:"message,omitempty"`
	UID     string `json:"uid,omitempty"`
}
