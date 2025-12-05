package app

// ErrMsg represents an error that occurred during an operation
type ErrMsg struct {
	Err error
}

func (e ErrMsg) Error() string {
	return e.Err.Error()
}

// LoadingMsg indicates a loading state
type LoadingMsg struct {
	Loading bool
}

// RefreshMsg signals a refresh of the current view
type RefreshMsg struct{}
