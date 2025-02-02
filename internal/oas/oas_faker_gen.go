// Code generated by ogen, DO NOT EDIT.

package oas

import (
	"time"
)

// SetFake set fake values.
func (s *Application) SetFake() {
	{
		{
			s.Name = "string"
		}
	}
	{
		{
			s.Namespace = "string"
		}
	}
}

// SetFake set fake values.
func (s *ApplicationList) SetFake() {
	var unwrapped []Application
	{
		unwrapped = nil
		for i := 0; i < 0; i++ {
			var elem Application
			{
				elem.SetFake()
			}
			unwrapped = append(unwrapped, elem)
		}
	}
	*s = ApplicationList(unwrapped)
}

// SetFake set fake values.
func (s *ApplicationSummary) SetFake() {
	{
		{
			s.Name = "string"
		}
	}
	{
		{
			s.Namespace = "string"
		}
	}
	{
		{
			s.Pods = nil
			for i := 0; i < 0; i++ {
				var elem Pod
				{
					elem.SetFake()
				}
				s.Pods = append(s.Pods, elem)
			}
		}
	}
}

// SetFake set fake values.
func (s *Error) SetFake() {
	{
		{
			s.ErrorMessage = "string"
		}
	}
	{
		{
			s.TraceID.SetFake()
		}
	}
	{
		{
			s.SpanID.SetFake()
		}
	}
}

// SetFake set fake values.
func (s *Health) SetFake() {
	{
		{
			s.Status = "string"
		}
	}
	{
		{
			s.Version = "string"
		}
	}
	{
		{
			s.Commit = "string"
		}
	}
	{
		{
			s.BuildDate = time.Now()
		}
	}
}

// SetFake set fake values.
func (s *OptSpanID) SetFake() {
	var elem SpanID
	{
		elem.SetFake()
	}
	s.SetTo(elem)
}

// SetFake set fake values.
func (s *OptTraceID) SetFake() {
	var elem TraceID
	{
		elem.SetFake()
	}
	s.SetTo(elem)
}

// SetFake set fake values.
func (s *Pod) SetFake() {
	{
		{
			s.Name = "string"
		}
	}
	{
		{
			s.Namespace = "string"
		}
	}
	{
		{
			s.Status = "string"
		}
	}
	{
		{
			s.Resources.SetFake()
		}
	}
}

// SetFake set fake values.
func (s *PodResources) SetFake() {
	{
		{
			s.CPUUsageTotalMillicores = float64(0)
		}
	}
	{
		{
			s.MemUsageTotalBytes = int64(0)
		}
	}
}

// SetFake set fake values.
func (s *SpanID) SetFake() {
	var unwrapped string
	{
		unwrapped = "string"
	}
	*s = SpanID(unwrapped)
}

// SetFake set fake values.
func (s *TraceID) SetFake() {
	var unwrapped string
	{
		unwrapped = "string"
	}
	*s = TraceID(unwrapped)
}
