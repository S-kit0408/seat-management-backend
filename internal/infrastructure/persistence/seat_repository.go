package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"seat-management-backend/internal/domain/entity"
	"seat-management-backend/internal/domain/repository"

	"gorm.io/gorm"
)

type seatRepository struct {
	db *gorm.DB
}

func NewSeatRepository(db *gorm.DB) repository.SeatRepository {
	return &seatRepository{db: db}
}

func (r *seatRepository) Create(ctx context.Context, seat *entity.Seat) error {
	return r.db.WithContext(ctx).Create(seat).Error
}

func (r *seatRepository) FindByID(ctx context.Context, id string) (*entity.Seat, error) {
	var seat entity.Seat
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&seat).Error
	if err != nil {
		// レコードが見つからない場合はカスタムエラーを返す
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, entity.ErrSeatNotFound
		}
		return nil, err
	}
	return &seat, nil
}

func (r *seatRepository) FindBySeatNumber(ctx context.Context, seatNumber string) (*entity.Seat, error) {
	var seat entity.Seat
	err := r.db.WithContext(ctx).Where("seat_number = ?", seatNumber).First(&seat).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, entity.ErrSeatNotFound
		}
		return nil, err
	}
	return &seat, nil
}

func (r *seatRepository) Update(ctx context.Context, seat *entity.Seat) error {
	return r.db.WithContext(ctx).Save(seat).Error
}

func (r *seatRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&entity.Seat{}, "id = ?", id).Error
}

func (r *seatRepository) List(ctx context.Context, limit, offset int) ([]*entity.Seat, error) {
	var seats []*entity.Seat
	err := r.db.WithContext(ctx).
		Limit(limit).
		Offset(offset).
		Order("seat_number ASC"). // 座席番号順にソート
		Find(&seats).Error
	return seats, err
}

func (r *seatRepository) FindAll(ctx context.Context) ([]*entity.Seat, error) {
	var seats []*entity.Seat
	err := r.db.WithContext(ctx).
		Order("seat_number ASC").
		Find(&seats).Error
	return seats, err
}

func (r *seatRepository) FindActive(ctx context.Context) ([]*entity.Seat, error) {
	var seats []*entity.Seat
	err := r.db.WithContext(ctx).
		Where("is_active = ?", true).
		Order("seat_number ASC").
		Find(&seats).Error
	return seats, err
}

func (r *seatRepository) FindActiveWithFloor(ctx context.Context) ([]*entity.Seat, error) {
	var seats []*entity.Seat
	err := r.db.WithContext(ctx).
		Where("is_active = ? AND floor_id IS NOT NULL", true).
		Order("seat_number ASC").
		Find(&seats).Error
	return seats, err
}

func (r *seatRepository) FindByFloorID(ctx context.Context, floorID string) ([]*entity.Seat, error) {
	var seats []*entity.Seat
	err := r.db.WithContext(ctx).
		Where("floor_id = ?", floorID).
		Order("seat_number ASC").
		Find(&seats).Error
	return seats, err
}

func (r *seatRepository) FindBySpaceID(ctx context.Context, spaceID string) ([]*entity.Seat, error) {
	var seats []*entity.Seat
	err := r.db.WithContext(ctx).
		Where("space_id = ?", spaceID).
		Order("seat_number ASC").
		Find(&seats).Error
	return seats, err
}

func (r *seatRepository) FindUnassignedSeats(ctx context.Context) ([]*entity.Seat, error) {
	var seats []*entity.Seat
	err := r.db.WithContext(ctx).
		Where("floor_id IS NULL").
		Order("seat_number ASC").
		Find(&seats).Error
	return seats, err
}

// FindByKeywords は自由形式のキーワード配列で座席を検索
// attributes内の「free_attributes」キーの配列値を検索
// 戻り値: 全一致結果, 部分一致結果, エラー
func (r *seatRepository) FindByKeywords(
	ctx context.Context,
	keywords []string,
	floorID *string,
) (exactMatches []*entity.Seat, partialMatches []*entity.Seat, err error) {
	if len(keywords) == 0 {
		// キーワードなしの場合、フロアフィルターのみ
		query := r.db.WithContext(ctx).Where("is_active = ?", true)
		if floorID != nil && *floorID != "" {
			query = query.Where("floor_id = ?", *floorID)
		}
		var seats []*entity.Seat
		if err := query.Order("seat_number ASC").Find(&seats).Error; err != nil {
			return nil, nil, err
		}
		return seats, nil, nil
	}

	baseQuery := r.db.WithContext(ctx).Where("is_active = ?", true)
	if floorID != nil && *floorID != "" {
		baseQuery = baseQuery.Where("floor_id = ?", *floorID)
	}

	// すべての座席を取得（メモリ内でフィルター）
	var allSeats []*entity.Seat
	if err := baseQuery.Order("seat_number ASC").Find(&allSeats).Error; err != nil {
		return nil, nil, err
	}

	// メモリ内で全一致・部分一致をフィルター
	exactMatches = make([]*entity.Seat, 0)
	partialMatches = make([]*entity.Seat, 0)

	for _, seat := range allSeats {
		// attributesからfree_attributesを取得
		freeAttrs := r.extractFreeAttributes(seat)

		// 全一致チェック：すべてのキーワードを含む
		if r.containsAllKeywords(freeAttrs, keywords) {
			exactMatches = append(exactMatches, seat)
		} else if r.containsAnyKeyword(freeAttrs, keywords) {
			// 部分一致チェック：少なくとも1つのキーワードを含む（全一致を除く）
			partialMatches = append(partialMatches, seat)
		}
	}

	return exactMatches, partialMatches, nil
}

// extractFreeAttributes はattributesからfree_attributes配列を取得
func (r *seatRepository) extractFreeAttributes(seat *entity.Seat) []string {
	if len(seat.Attributes) == 0 {
		return []string{}
	}

	// JSON からfree_attributes キーを抽出
	var attrs map[string]interface{}
	if err := json.Unmarshal(seat.Attributes, &attrs); err != nil {
		return []string{}
	}

	freeAttrs, ok := attrs["free_attributes"]
	if !ok {
		return []string{}
	}

	// インターフェースから[]stringに変換
	var result []string
	switch v := freeAttrs.(type) {
	case []interface{}:
		for _, item := range v {
			if str, ok := item.(string); ok {
				result = append(result, str)
			}
		}
	}

	return result
}

// containsAllKeywords はすべてのキーワードを含むかチェック
func (r *seatRepository) containsAllKeywords(attributes, keywords []string) bool {
	for _, kw := range keywords {
		found := false
		for _, attr := range attributes {
			if strings.Contains(attr, kw) || strings.Contains(kw, attr) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// containsAnyKeyword は少なくとも1つのキーワードを含むかチェック
func (r *seatRepository) containsAnyKeyword(attributes, keywords []string) bool {
	for _, kw := range keywords {
		for _, attr := range attributes {
			if strings.Contains(attr, kw) || strings.Contains(kw, attr) {
				return true
			}
		}
	}
	return false
}

// FindByAttributes は属性パラメータでフィルター（互換性保持用）
func (r *seatRepository) FindByAttributes(ctx context.Context, params map[string]interface{}) ([]*entity.Seat, error) {
	query := r.db.WithContext(ctx).Where("is_active = ?", true)

	// has_power_outlet フィルター
	if hasPower, ok := params["has_power_outlet"]; ok && hasPower != nil {
		if powerBool, ok := hasPower.(bool); ok && powerBool {
			query = query.Where("attributes->>'has_power_outlet' = 'true'")
		}
	}

	// work_style フィルター
	if workStyle, ok := params["work_style"]; ok && workStyle != nil {
		if styleStr, ok := workStyle.(string); ok && styleStr != "" {
			query = query.Where("attributes->>'work_style' = ?", styleStr)
		}
	}

	// is_window_seat フィルター
	if isWindow, ok := params["is_window_seat"]; ok && isWindow != nil {
		if windowBool, ok := isWindow.(bool); ok && windowBool {
			query = query.Where("attributes->>'is_window_seat' = 'true'")
		}
	}

	// floor_id フィルター
	if floorID, ok := params["floor_id"]; ok && floorID != nil {
		if floorStr, ok := floorID.(string); ok && floorStr != "" {
			query = query.Where("floor_id = ?", floorStr)
		}
	}

	var seats []*entity.Seat
	if err := query.Order("seat_number ASC").Find(&seats).Error; err != nil {
		return nil, err
	}

	return seats, nil
}
