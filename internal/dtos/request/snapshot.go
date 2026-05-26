package request

import "encoding/json"

type CommitSnapshot struct {
	Project                 *ProjectSnapshot          `json:"project" validate:"omitempty"`
	EditorFeatureCollection *FeatureCollection        `json:"editor_feature_collection" validate:"omitempty"`
	Entities                []*EntitySnapshot         `json:"entities" validate:"omitempty,dive"`
	Geometries              []*GeometrySnapshot       `json:"geometries" validate:"omitempty,dive"`
	Wikis                   []*WikiSnapshot           `json:"wikis" validate:"omitempty,dive"`
	GeometryEntity          []*GeometryEntitySnapshot `json:"geometry_entity" validate:"omitempty,dive"`
	EntityWiki              []*EntityWikiLinkSnapshot `json:"entity_wiki" validate:"omitempty,dive"`
	Replays                 []*BattleReplaySnapshot   `json:"replays" validate:"omitempty,dive"`
}

type ProjectSnapshot struct {
	ID    string `json:"id" validate:"required,uuidv7"`
	Title string `json:"title" validate:"required"`
}

type BattleReplaySnapshot struct {
	ID                string          `json:"id" validate:"required,uuidv7"`
	GeometryID        string          `json:"geometry_id" validate:"required,uuidv7"`
	TargetGeometryIDs []string        `json:"target_geometry_ids" validate:"omitempty,dive,uuidv7"`
	Detail            json.RawMessage `json:"detail" validate:"omitempty"`
}

type FeatureCollection struct {
	Type     string     `json:"type" validate:"required,eq=FeatureCollection"`
	Features []*Feature `json:"features" validate:"required,dive"`
}

type Feature struct {
	Type       string             `json:"type" validate:"required,eq=Feature"`
	Properties *FeatureProperties `json:"properties" validate:"required"`
	Geometry   json.RawMessage    `json:"geometry" validate:"required"`
}

type FeatureProperties struct {
	ID                    any                      `json:"id" validate:"required"`
	Source                string                   `json:"source" validate:"omitempty,oneof=inline ref"`
	Type                  string                   `json:"type" validate:"omitempty"`
	GeometryPreset        string                   `json:"geometry_preset" validate:"omitempty"`
	TimeStart             *float64                 `json:"time_start" validate:"omitempty"`
	TimeEnd               *float64                 `json:"time_end" validate:"omitempty"`
	BoundWith             *string                  `json:"bound_with" validate:"omitempty"`
	EntityID              string                   `json:"entity_id" validate:"omitempty,uuidv7"`
	EntityIDs             []string                 `json:"entity_ids" validate:"omitempty,dive,uuidv7"`
	EntityName            string                   `json:"entity_name"`
	EntityNames           []string                 `json:"entity_names" validate:"omitempty"`
	EntityTypeID          string                   `json:"entity_type_id" validate:"omitempty,uuidv7"`
	PointLabel            *string                  `json:"point_label" validate:"omitempty"`
	LineLabel             *string                  `json:"line_label" validate:"omitempty"`
	PolygonLabel          *string                  `json:"polygon_label" validate:"omitempty"`
	EntityLabelCandidates []*EntityLabelCandidate `json:"entity_label_candidates" validate:"omitempty,dive"`
}

type EntityLabelCandidate struct {
	ID        string   `json:"id" validate:"required,uuidv7"`
	Name      string   `json:"name" validate:"required"`
	TimeStart *float64 `json:"time_start" validate:"omitempty"`
	TimeEnd   *float64 `json:"time_end" validate:"omitempty"`
}

type EntitySnapshot struct {
	ID            string   `json:"id" validate:"required,uuidv7"`
	Source        string   `json:"source" validate:"omitempty,oneof=inline ref"`
	Operation     string   `json:"operation" validate:"omitempty,oneof=create update delete reference"`
	Name          string   `json:"name" validate:"omitempty"`
	Slug          *string  `json:"slug" validate:"omitempty,slug"`
	Description   string   `json:"description" validate:"omitempty"`
	Status        *int     `json:"status" validate:"omitempty,oneof=0 1"`
	TimeStart     *float64 `json:"time_start" validate:"omitempty"`
	TimeEnd       *float64 `json:"time_end" validate:"omitempty"`
	BaseUpdatedAt string   `json:"base_updated_at" validate:"omitempty"`
	BaseHash      string   `json:"base_hash" validate:"omitempty"`
}

type GeometrySnapshot struct {
	ID            string          `json:"id" validate:"required,uuidv7"`
	Source        string          `json:"source" validate:"omitempty,oneof=inline ref"`
	Operation     string          `json:"operation" validate:"omitempty,oneof=create update delete reference"`
	Type          string          `json:"type" validate:"omitempty"`
	DrawGeometry  json.RawMessage `json:"draw_geometry" validate:"omitempty"`
	Geometry      json.RawMessage `json:"geometry" validate:"omitempty"`
	BoundWith     *string         `json:"bound_with" validate:"omitempty"`
	TimeStart     *float64        `json:"time_start" validate:"omitempty"`
	TimeEnd       *float64        `json:"time_end" validate:"omitempty"`
	BBox          *BBox           `json:"bbox" validate:"omitempty"`
	BaseUpdatedAt string          `json:"base_updated_at" validate:"omitempty"`
	BaseHash      string          `json:"base_hash" validate:"omitempty"`
}

type BBox struct {
	MinLng float64 `json:"min_lng" validate:"required"`
	MinLat float64 `json:"min_lat" validate:"required"`
	MaxLng float64 `json:"max_lng" validate:"required"`
	MaxLat float64 `json:"max_lat" validate:"required"`
}

type GeometryEntitySnapshot struct {
	GeometryID    string `json:"geometry_id" validate:"required,uuidv7"`
	EntityID      string `json:"entity_id" validate:"required,uuidv7"`
	Operation     string `json:"operation" validate:"omitempty,oneof=reference delete binding"`
	BaseLinksHash string `json:"base_links_hash" validate:"omitempty"`
}

type WikiSnapshot struct {
	ID        string  `json:"id" validate:"required,uuidv7"`
	Source    string  `json:"source" validate:"omitempty,oneof=inline ref"`
	Operation string  `json:"operation" validate:"omitempty,oneof=create update delete reference"`
	Title     string  `json:"title" validate:"required"`
	Slug      *string `json:"slug" validate:"omitempty,slug"`
	Doc       string  `json:"doc" validate:"omitempty"`
	UpdatedAt string  `json:"updated_at" validate:"omitempty"`
}

type EntityWikiLinkSnapshot struct {
	EntityID  string `json:"entity_id" validate:"required,uuidv7"`
	WikiID    string `json:"wiki_id" validate:"required,uuidv7"`
	Operation string `json:"operation" validate:"omitempty,oneof=reference delete binding"`
	IsDeleted *int   `json:"is_deleted" validate:"omitempty,oneof=0 1"`
}
