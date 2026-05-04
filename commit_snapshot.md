# Commit Snapshot (`commits.snapshot_json`) - Chuẩn Hiện Tại (FrontEndAdmin)

Tài liệu này mô tả **commit snapshot** được `FrontEndAdmin` tạo ra khi bấm **Commit** trong `/editor`, và được lưu vào `BackEndGo.commits.snapshot_json` (JSONB).

Nguồn tham chiếu trong code:

- Type snapshot: `FrontEndAdmin/src/uhm/types/sections.ts` (`EditorSnapshot`)
- Build snapshot: `FrontEndAdmin/src/uhm/lib/editor/snapshot/editorSnapshot.ts` (`buildEditorSnapshot`)

## 1) Tổng Quan Schema

Snapshot hiện tại:

- Không có `schema_version`.
- Không lưu `section` (entity trong DB là `projects`; project được xác định bằng context record `commits.project_id`).
- Không dùng `ref:{id}` nữa: **`id` là canonical**, `source:"ref"` nghĩa là tham chiếu theo `id`.

```ts
export type CommitSnapshot = {
  editor_feature_collection?: FeatureCollection;

  entities?: EntitySnapshot[];
  geometries?: GeometrySnapshot[];
  wikis?: WikiSnapshot[];

  geometry_entity?: GeometryEntitySnapshot[]; // geometry ↔ entity (many-to-many)
  entity_wiki?: EntityWikiLinkSnapshot[];     // entity ↔ wiki
};
```

## 1.1 Type đầy đủ (TypeScript)

Đây là bản type "đúng để BEGo implement chuyển đổi snapshot → DB". FE có thể gửi thêm field legacy (xem mục 6), nhưng BE nên normalize theo các type dưới đây.

```ts
// ---- GeoJSON ----

export type Geometry =
  | { type: "Point"; coordinates: [number, number] }
  | { type: "MultiPoint"; coordinates: [number, number][] }
  | { type: "LineString"; coordinates: [number, number][] }
  | { type: "MultiLineString"; coordinates: [number, number][][] }
  | { type: "Polygon"; coordinates: [number, number][][] }
  | { type: "MultiPolygon"; coordinates: [number, number][][][] };

export type FeatureId = string | number; // FE hiện dùng UUIDv7 string

export type FeatureProperties = {
  id: FeatureId;
  type?: string | null;
  geometry_preset?: string | null;
  time_start?: number | null;
  time_end?: number | null;
  binding?: string[]; // entity ids used as "binding filter"

  // Legacy UI fields. FE persist snapshot hiện tại KHONG gửi các field này,
  // nhưng BE nên ignore nếu gặp trong snapshot cũ:
  entity_id?: string | null;
  entity_ids?: string[];
  entity_name?: string | null;
  entity_names?: string[];
  entity_type_id?: string | null;
};

export type Feature = {
  type: "Feature";
  properties: FeatureProperties;
  geometry: Geometry;
};

export type FeatureCollection = {
  type: "FeatureCollection";
  features: Feature[];
};

// ---- Snapshot rows ----

export type SnapshotSource = "inline" | "ref";

export type EntitySnapshotOperation = "create" | "update" | "delete" | "reference";
export type GeometrySnapshotOperation = "create" | "update" | "delete" | "reference";
export type WikiSnapshotOperation = "create" | "update" | "delete" | "reference";

export type EntitySnapshot = {
  id: string; // UUIDv7 string (canonical)
  source: SnapshotSource;
  operation?: EntitySnapshotOperation;
  name?: string;
  slug?: string | null;
  description?: string | null;
  type_id?: string | null;
  status?: number | null;
  base_updated_at?: string;
  base_hash?: string;
};

export type GeometrySnapshot = {
  id: string; // UUIDv7 string (canonical)
  source: SnapshotSource;
  operation?: GeometrySnapshotOperation;

  // Present when source:"inline" (draft features)
  type?: string | null;
  draw_geometry?: Geometry;
  binding?: string[];
  time_start?: number | null;
  time_end?: number | null;
  bbox?: {
    min_lng: number;
    min_lat: number;
    max_lng: number;
    max_lat: number;
  } | null;

  base_updated_at?: string;
  base_hash?: string;
};

export type WikiSnapshot = {
  id: string; // UUIDv7 string (canonical)
  source: SnapshotSource;
  operation?: WikiSnapshotOperation;
  title: string;
  doc: unknown; // tiptap JSON (inline) hoặc null (ref)
  updated_at?: string;
};

// ---- Join tables ----

export type GeometryEntitySnapshot = {
  geometry_id: string;
  entity_id: string;
  base_links_hash?: string;
};

export type EntityWikiLinkSnapshot = {
  entity_id: string;
  wiki_id: string;
  // If missing, BE should treat as "reference" (active link) for backwards-compat.
  operation?: "reference" | "delete";
};

// ---- Root ----

export type CommitSnapshot = {
  editor_feature_collection?: FeatureCollection;
  entities?: EntitySnapshot[];
  geometries?: GeometrySnapshot[];
  wikis?: WikiSnapshot[];
  geometry_entity?: GeometryEntitySnapshot[];
  entity_wiki?: EntityWikiLinkSnapshot[];
};
```

## 2) Quy Ước `source` và `operation`

### 2.1 `source` (bắt buộc)

`source` bắt buộc là một trong:

- `inline`: dữ liệu được embed trong snapshot_json.
- `ref`: dữ liệu là tham chiếu (theo `id`), cần fetch bên ngoài nếu muốn đầy đủ.

FE hiện tại luôn ghi `source` cho `entities[]`, `geometries[]`, `wikis[]`.

### 2.2 `operation` (tùy chọn)

`operation` là tùy chọn. Khi **không có** `operation` thì hiểu là:

- row được đưa vào snapshot như **project context** (hoặc không đổi trong commit này),
- commit này không sửa record, và cũng không cần đánh dấu là `"reference"` để làm “đầu mối nối”.

`operation` có thể xuất hiện ở:

- `entities[].operation`: `create` | `update` | `delete` | `reference`
- `geometries[].operation`: `create` | `update` | `delete` | `reference`
- `wikis[].operation`: `create` | `update` | `delete` | `reference`

`geometry_entity[]` không có `operation` (join table state).

`entity_wiki[]` dùng `operation:"reference"|"delete"` để biểu diễn link/unlink **trong snapshot** (không phải delete trong DB).

## 3) Ý Nghĩa Từng Phần

### 3.1 `editor_feature_collection`

GeoJSON `FeatureCollection` là nguồn để:

- render map trong editor,
- làm cơ sở build `geometries[]` và join table `geometry_entity[]`.

Lưu ý quan trọng:

- Snapshot persist **không lưu** các field entity denormalize trên `feature.properties`:
  `entity_id/entity_ids/entity_name/entity_names/entity_type_id`.
- Quan hệ geometry ↔ entity nằm ở `geometry_entity[]`.
- Khi load commit vào editor, FE có thể rehydrate `entity_ids/entity_id` lên features từ `geometry_entity[]` để UI hoạt động, nhưng đó không phải dữ liệu persist.

### 3.2 `entities[]`

`entities[]` là danh sách entity liên quan tới project/commit. Mỗi row có `source` và có thể có/không có `operation`.

FE build `entities[]` từ:

1. Pending entities tạo mới trong editor:
`source:"inline"`, `operation:"create"`.

2. Entity được user “pin” vào project từ search (không gắn geometry, không link wiki):
`source:"ref"`, không có `operation`.

3. Entities xuất hiện trong `geometry_entity[]`:
`source:"ref"`, `operation:"reference"`.

4. Entities xuất hiện trong `entity_wiki[]`:
`source:"ref"`, `operation:"reference"`.

### 3.3 `geometries[]`

Mỗi `Feature` trong `editor_feature_collection.features[]` sinh 1 `GeometrySnapshot` row:

- `id = String(feature.properties.id)`
- `source:"inline"`
- `draw_geometry = feature.geometry`
- kèm `type`, `binding`, `time_start/time_end`, `bbox` (nếu tính được)

`operation` cho geometry:

- `create`: feature mới
- `update`: feature thay đổi
- (không có `operation`): feature không đổi (không delta trong commit)

Nếu feature bị xoá khỏi draft, FE thêm 1 delete row:

```json
{ "id": "g_1", "source": "ref", "operation": "delete" }
```

Lưu ý: geometry `operation:"delete"` **không xuất hiện trên map**, vì map render theo `editor_feature_collection.features[]`.

Gợi ý cho BE khi apply vào DB:

- Có thể coi `editor_feature_collection` là state hiện tại để render/map.
- `geometries[]` là "rows + deltas": sẽ có 1 row cho mỗi feature đang tồn tại trong draft (có/không có `operation`), và có thể có thêm các row `operation:"delete"` để xoá geometry khỏi project state.

### 3.4 `geometry_entity[]` (join table Geometry ↔ Entity)

Join table many-to-many giữa geometry và entity. Mỗi cặp geometry↔entity là một row:

```ts
{ geometry_id: string; entity_id: string }
```

### 3.5 `wikis[]`

Danh sách wiki của project tại thời điểm commit:

- Wiki tạo mới: `source:"inline"`, `operation:"create"`, `doc` là tiptap JSON.
- Wiki sửa: `source:"inline"`, `operation:"update"`, `doc` là tiptap JSON.
- Wiki không đổi: thường không có `operation`.
- Wiki add từ search (wiki đã có trong DB): `source:"ref"`, `operation:"reference"`, `doc` có thể là `null`.

### 3.6 `entity_wiki[]` (join table Entity ↔ Wiki)

```ts
export type EntityWikiLinkSnapshot = {
  entity_id: string;
  wiki_id: string;
  operation?: "reference" | "delete";
};
```

Toggle link trong UI:

- Tick checkbox: `{ operation: "reference" }`
- Untick checkbox: `{ operation: "delete" }`

## 4) Ví Dụ JSON (rút gọn)

```json
{
  "editor_feature_collection": {
    "type": "FeatureCollection",
    "features": [
      {
        "type": "Feature",
        "properties": {
          "id": "g_1",
          "type": "city",
          "time_start": 1200,
          "time_end": 1300,
          "binding": []
        },
        "geometry": { "type": "Point", "coordinates": [105.8, 21.0] }
      }
    ]
  },
  "entities": [
    { "id": "e_2", "source": "ref", "name": "Pinned Entity" },
    { "id": "e_1", "source": "ref", "operation": "reference", "name": "Ha Noi", "type_id": "city", "status": 1 }
  ],
  "geometries": [
    {
      "id": "g_1",
      "source": "inline",
      "operation": "update",
      "type": "city",
      "draw_geometry": { "type": "Point", "coordinates": [105.8, 21.0] },
      "binding": [],
      "time_start": 1200,
      "time_end": 1300,
      "bbox": { "min_lng": 105.8, "min_lat": 21.0, "max_lng": 105.8, "max_lat": 21.0 }
    }
  ],
  "geometry_entity": [
    { "geometry_id": "g_1", "entity_id": "e_1" }
  ],
  "wikis": [
    {
      "id": "w_inline_1",
      "source": "inline",
      "operation": "create",
      "title": "Overview",
      "doc": { "type": "doc", "content": [{ "type": "paragraph" }] }
    },
    {
      "id": "019d...wiki_from_db",
      "source": "ref",
      "operation": "reference",
      "title": "Existing Wiki (DB)",
      "doc": null
    }
  ],
  "entity_wiki": [
    { "entity_id": "e_1", "wiki_id": "w_inline_1", "operation": "reference" }
  ]
}
```

## 5) Notes Cho BackEnd (Normalize + Compat)

BE nên normalize trước khi convert snapshot → DB:

- Ignore toàn bộ field entity denormalize trên `feature.properties` (nếu có): `entity_id/entity_ids/entity_name/entity_names/entity_type_id`. Quan hệ geometry↔entity lấy từ `geometry_entity[]`.
- `entity_wiki[].operation`:
  - `"reference"`: link active
  - `"delete"`: link removed trong snapshot
  - missing: treat as `"reference"` (compat)

## 6) Legacy Compatibility (nếu gặp snapshot cũ)

FE đã từng gửi các field legacy; BE có thể gặp nếu đang xử lý commit cũ:

- `entity_wikis` (plural) thay vì `entity_wiki` (singular): treat như nhau.
- `ref:{id}` trong `entities/geometries/wikis`: ignore (id canonical).
- `is_deleted` trong join table entity↔wiki: map sang `operation:"delete"` khi `is_deleted==1`, ngược lại `"reference"`.
