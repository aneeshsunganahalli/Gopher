package types

// AddMetadata adds a key-value pair to job metadata
func (j *Job) AddMetadata(key string, value interface{}) {
	if j.Metadata == nil {
		j.Metadata = make(JobMetadata)
	}
	j.Metadata[key] = value
}

// GetMetadata retrieves a value from job metadata
func (j *Job) GetMetadata(key string) (interface{}, bool) {
	if j.Metadata == nil {
		return nil, false
	}
	val, ok := j.Metadata[key]
	return val, ok
}

// SetPriority sets the job priority
func (j *Job) SetPriority(priority string) {
	j.AddMetadata("priority", priority)
}

// GetPriority gets the job priority, defaulting to "normal" if not set
func (j *Job) GetPriority() string {
	val, ok := j.GetMetadata("priority")
	if !ok {
		return "normal"
	}

	priority, ok := val.(string)
	if !ok {
		return "normal"
	}

	return priority
}
