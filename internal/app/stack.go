package app

import "github.com/espen/lazylab/internal/ui/views"

// ViewStack manages the navigation history
type ViewStack struct {
	views []views.View
}

// NewViewStack creates an empty view stack
func NewViewStack() *ViewStack {
	return &ViewStack{
		views: make([]views.View, 0),
	}
}

// Push adds a view to the top of the stack
func (s *ViewStack) Push(v views.View) {
	s.views = append(s.views, v)
}

// Pop removes and returns the top view
func (s *ViewStack) Pop() views.View {
	if len(s.views) == 0 {
		return nil
	}
	v := s.views[len(s.views)-1]
	s.views = s.views[:len(s.views)-1]
	return v
}

// Current returns the top view without removing it
func (s *ViewStack) Current() views.View {
	if len(s.views) == 0 {
		return nil
	}
	return s.views[len(s.views)-1]
}

// Len returns the number of views in the stack
func (s *ViewStack) Len() int {
	return len(s.views)
}

// Breadcrumbs returns the titles of all views for navigation display
func (s *ViewStack) Breadcrumbs() []string {
	titles := make([]string, len(s.views))
	for i, v := range s.views {
		titles[i] = v.Title()
	}
	return titles
}
