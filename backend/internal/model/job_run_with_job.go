package model

// JobRunWithJob is a view model for job run history with job metadata.
type JobRunWithJob struct {
    JobRun
    JobTypeKey   string `gorm:"column:job_type_key" json:"job_type_key"`
    JobActionKey string `gorm:"column:job_action_key" json:"job_action_key"`
}
