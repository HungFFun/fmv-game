// Export — tái dựng StoryFile (format authoring, tham chiếu theo CODE) từ DB cho 1 model.
// Đối xứng với Load/Replace: Export → sửa → Replace phải cho ra DB tương đương (round-trip).
// media_assets không lưu "key" authoring nên ở đây sinh key tổng hợp (m_<id>); do key chỉ là
// ref nội bộ, round-trip vẫn đúng dù chuỗi key khác bản gốc.
package seed

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"fmv-game/backend/internal/store"
)

func rawIfSet(s string) json.RawMessage {
	if s == "" || s == "{}" {
		return nil
	}
	return json.RawMessage(s)
}

// Export dựng StoryFile đầy đủ cho model (dùng cho GET .../content của editor).
func Export(st *store.Store, modelID int64) (*StoryFile, error) {
	m, err := st.Model(modelID)
	if err != nil {
		return nil, err
	}
	sf := &StoryFile{}
	sf.Model = ModelDef{
		Code: m.Code, DisplayName: m.DisplayName, Avatar: m.Avatar.String,
		Age: int(m.Age.Int64), Birthday: m.Birthday.String, Relationship: m.Relationship.String,
		Occupation: m.Occupation.String, HeightCm: int(m.HeightCm.Int64), WeightKg: int(m.WeightKg.Int64),
		Family: m.Family.String, Bio: m.Bio.String,
	}

	// story slug/title
	if err := st.DB.QueryRow(`SELECT slug, title FROM stories WHERE model_id=?`, modelID).
		Scan(&sf.Story.Slug, &sf.Story.Title); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	chars, err := st.Characters(modelID)
	if err != nil {
		return nil, err
	}
	for _, c := range chars {
		sf.Characters = append(sf.Characters, CharacterDef{Code: c.Code, DisplayName: c.DisplayName, Archetype: c.Archetype})
	}

	// media: gom theo id được tham chiếu → sinh key tổng hợp.
	mediaKey := map[int64]string{}
	addMedia := func(id int64) (string, error) {
		if id == 0 {
			return "", nil
		}
		if k, ok := mediaKey[id]; ok {
			return k, nil
		}
		md, err := st.Media(id)
		if err != nil {
			return "", err
		}
		key := fmt.Sprintf("m_%d", id)
		mediaKey[id] = key
		sf.Media = append(sf.Media, MediaDef{Key: key, Kind: md.Kind, File: md.StorageKey, DurationMs: int(md.DurationMs.Int64)})
		return key, nil
	}

	chapters, err := st.Chapters(modelID)
	if err != nil {
		return nil, err
	}

	// Pass 1: scene id → code toàn model (cạnh có thể liên-chapter).
	sceneCode := map[int64]string{}
	type chData struct {
		ch     store.Chapter
		videos []store.ChapterVideo
		scenes []store.Scene
	}
	chds := make([]chData, 0, len(chapters))
	for _, ch := range chapters {
		vids, err := st.ChapterVideos(ch.ID)
		if err != nil {
			return nil, err
		}
		scs, err := st.Scenes(ch.ID)
		if err != nil {
			return nil, err
		}
		for _, sc := range scs {
			sceneCode[sc.ID] = sc.Code
		}
		chds = append(chds, chData{ch, vids, scs})
	}
	codeOf := func(n sql.NullInt64) string {
		if n.Valid {
			return sceneCode[n.Int64]
		}
		return ""
	}

	// Pass 2: dựng chapter/video/scene/choice.
	for _, cd := range chds {
		videoCode := map[int64]string{}
		videos := make([]VideoDef, 0, len(cd.videos))
		for _, v := range cd.videos {
			key, err := addMedia(v.MediaID.Int64)
			if err != nil {
				return nil, err
			}
			videoCode[v.ID] = v.Code
			videos = append(videos, VideoDef{
				Code: v.Code, Idx: v.Idx, Title: v.Title, Media: key,
				Poster: v.Poster.String, DurationMs: int(v.DurationMs.Int64),
			})
		}
		scenes := make([]SceneDef, 0, len(cd.scenes))
		for _, sc := range cd.scenes {
			sd := SceneDef{Code: sc.Code, Type: sc.Type, Checkpoint: sc.IsCheckpoint, OnEnter: rawIfSet(sc.OnEnterRaw)}
			if sc.VideoID.Valid {
				sd.Video = videoCode[sc.VideoID.Int64]
			}
			switch sc.Type {
			case "linear":
				sd.Next = codeOf(sc.NextSceneID)
			case "choice":
				choices, err := st.ChoicesForScene(sc.ID)
				if err != nil {
					return nil, err
				}
				for _, c := range choices {
					cdf := ChoiceDef{
						Label: c.Label, Next: codeOf(c.NextSceneID),
						Condition: rawIfSet(c.ConditionRaw), Effects: rawIfSet(c.EffectsRaw),
					}
					if c.TimerMs.Valid {
						cdf.TimerMs = int(c.TimerMs.Int64)
					}
					if c.DefaultChoiceID.Valid && c.DefaultChoiceID.Int64 == c.ID {
						cdf.Default = true
					}
					if c.Hotspot.Valid && c.Hotspot.String != "" {
						var h HotspotDef
						if json.Unmarshal([]byte(c.Hotspot.String), &h) == nil {
							cdf.Hotspot = &h
						}
					}
					sd.Choices = append(sd.Choices, cdf)
				}
			}
			scenes = append(scenes, sd)
		}
		chDef := ChapterDef{
			Idx: cd.ch.Idx, Title: cd.ch.Title, IsFree: cd.ch.IsFree, PriceCents: cd.ch.PriceCents,
			SKU: cd.ch.SKU.String, Poster: cd.ch.Poster.String, Entry: codeOf(cd.ch.EntrySceneID),
			Map: rawIfSet(cd.ch.MapJSON), Videos: videos, Scenes: scenes,
		}
		sf.Chapters = append(sf.Chapters, chDef)
	}

	// endings (theo model).
	rows, err := st.DB.Query(`SELECT scene_id, code, title, COALESCE(rank,'') FROM endings WHERE model_id=? ORDER BY id`, modelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var sceneID int64
		var e EndingDef
		if err := rows.Scan(&sceneID, &e.Code, &e.Title, &e.Rank); err != nil {
			return nil, err
		}
		e.Scene = sceneCode[sceneID]
		sf.Endings = append(sf.Endings, e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// gallery.
	gitems, err := st.GalleryItems(modelID)
	if err != nil {
		return nil, err
	}
	for _, g := range gitems {
		key, err := addMedia(g.MediaID)
		if err != nil {
			return nil, err
		}
		sf.Gallery = append(sf.Gallery, GalleryDef{
			Title: g.Title.String, Media: key, UnlockScene: codeOf(g.UnlockSceneID), IsBonus: g.IsBonus,
		})
	}

	return sf, nil
}
