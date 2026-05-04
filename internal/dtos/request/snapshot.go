package request

import "encoding/json"

type CommitSnapshot struct {
	EditorFeatureCollection *FeatureCollection        `json:"editor_feature_collection,omitempty" validate:"omitempty"`
	Entities                []*EntitySnapshot         `json:"entities,omitempty" validate:"omitempty,dive"`
	Geometries              []*GeometrySnapshot       `json:"geometries,omitempty" validate:"omitempty,dive"`
	Wikis                   []*WikiSnapshot           `json:"wikis,omitempty" validate:"omitempty,dive"`
	GeometryEntity          []*GeometryEntitySnapshot `json:"geometry_entity,omitempty" validate:"omitempty,dive"`
	EntityWiki              []*EntityWikiLinkSnapshot `json:"entity_wiki,omitempty" validate:"omitempty,dive"`
	EntityWikis             []*EntityWikiLinkSnapshot `json:"entity_wikis,omitempty" validate:"omitempty,dive"`
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
	ID             any      `json:"id" validate:"required"`
	Type           string   `json:"type,omitempty"`
	GeometryPreset string   `json:"geometry_preset,omitempty"`
	TimeStart      *float64 `json:"time_start,omitempty"`
	TimeEnd        *float64 `json:"time_end,omitempty"`
	Binding        []string `json:"binding,omitempty"`
	EntityID       string   `json:"entity_id,omitempty" validate:"omitempty,uuidv7"`
	EntityIDs      []string `json:"entity_ids,omitempty" validate:"omitempty,dive,uuidv7"`
	EntityName     string   `json:"entity_name,omitempty"`
	EntityNames    []string `json:"entity_names,omitempty"`
	EntityTypeID   string   `json:"entity_type_id,omitempty" validate:"omitempty,uuidv7"`
}

type EntitySnapshot struct {
	ID            string   `json:"id" validate:"required,uuidv7"`
	Source        string   `json:"source,omitempty" validate:"omitempty,oneof=inline ref"`
	Operation     string   `json:"operation,omitempty" validate:"omitempty,oneof=create update delete reference"`
	Name          string   `json:"name,omitempty"`
	Slug          *string  `json:"slug,omitempty"`
	Description   string   `json:"description,omitempty"`
	Status        *int     `json:"status,omitempty" validate:"omitempty,oneof=0 1"`
	TimeStart     *float64 `json:"time_start,omitempty"`
	TimeEnd       *float64 `json:"time_end,omitempty"`
	BaseUpdatedAt string   `json:"base_updated_at,omitempty"`
	BaseHash      string   `json:"base_hash,omitempty"`
}

type GeometrySnapshot struct {
	ID            string          `json:"id" validate:"required,uuidv7"`
	Source        string          `json:"source,omitempty" validate:"omitempty,oneof=inline ref"`
	Operation     string          `json:"operation,omitempty" validate:"omitempty,oneof=create update delete reference"`
	Type          string          `json:"type" validate:"required"`
	DrawGeometry  json.RawMessage `json:"draw_geometry,omitempty"`
	Binding       []string        `json:"binding,omitempty"`
	TimeStart     *float64        `json:"time_start,omitempty"`
	TimeEnd       *float64        `json:"time_end,omitempty"`
	BBox          *BBox           `json:"bbox,omitempty" validate:"omitempty"`
	BaseUpdatedAt string          `json:"base_updated_at,omitempty"`
	BaseHash      string          `json:"base_hash,omitempty"`
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
	BaseLinksHash string `json:"base_links_hash,omitempty"`
}

type WikiSnapshot struct {
	ID        string          `json:"id" validate:"required,uuidv7"`
	Source    string          `json:"source,omitempty" validate:"omitempty,oneof=inline ref"`
	Operation string          `json:"operation,omitempty" validate:"omitempty,oneof=create update delete reference"`
	Title     string          `json:"title" validate:"required"`
	Doc       string          `json:"doc,omitempty"`
	UpdatedAt string          `json:"updated_at,omitempty"`
}

type EntityWikiLinkSnapshot struct {
	EntityID  string `json:"entity_id" validate:"required,uuidv7"`
	WikiID    string `json:"wiki_id" validate:"required,uuidv7"`
	Operation string `json:"operation,omitempty" validate:"omitempty,oneof=reference delete"`
	IsDeleted *int   `json:"is_deleted,omitempty" validate:"omitempty,oneof=0 1"`
}
