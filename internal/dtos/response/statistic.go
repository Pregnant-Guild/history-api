package response

import "time"

type StatisticResponse struct {
	ID                string    `json:"id"`
	Date              string    `json:"date"`
	TotalUsers        int32     `json:"total_users"`
	TotalProjects     int32     `json:"total_projects"`
	TotalCommits      int32     `json:"total_commits"`
	TotalSubmissions  int32     `json:"total_submissions"`
	TotalMedias       int32     `json:"total_medias"`
	TotalWikis        int32     `json:"total_wikis"`
	TotalEntities     int32     `json:"total_entities"`
	TotalGeometries   int32     `json:"total_geometries"`
	TotalStorageBytes int64     `json:"total_storage_bytes"`
	NewUsers          int32     `json:"new_users"`
	NewProjects       int32     `json:"new_projects"`
	NewCommits        int32     `json:"new_commits"`
	NewSubmissions    int32     `json:"new_submissions"`
	NewMedias         int32     `json:"new_medias"`
	NewWikis          int32     `json:"new_wikis"`
	NewEntities       int32     `json:"new_entities"`
	NewGeometries     int32     `json:"new_geometries"`
	NewStorageBytes   int64     `json:"new_storage_bytes"`
	CreatedAt         time.Time `json:"created_at"`
}
