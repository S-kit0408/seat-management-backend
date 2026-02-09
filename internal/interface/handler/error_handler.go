package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"seat-management-backend/internal/domain/entity"
)

func RespondWithError(c *gin.Context, statusCode int, message string, code string, details ...map[string]interface{}) {
	response := ErrorResponse{
		Error: message,
		Code:  code,
	}
	if len(details) > 0 {
		response.Details = details[0]
	}
	c.JSON(statusCode, response)
}

func HandleUsecaseError(c *gin.Context, err error, defaultMessage string) {
	if err == nil {
		return
	}

	switch {
	case errors.Is(err, entity.ErrUserNotFound):
		RespondWithError(c, http.StatusNotFound, "ユーザーが見つかりません", CodeNotFound)
	case errors.Is(err, entity.ErrInvalidEmail):
		RespondWithError(c, http.StatusBadRequest, "無効なメールアドレスです", CodeValidation)
	case errors.Is(err, entity.ErrInvalidName):
		RespondWithError(c, http.StatusBadRequest, "無効な名前です", CodeValidation)
	case errors.Is(err, entity.ErrInvalidRole):
		RespondWithError(c, http.StatusBadRequest, "無効なロールです", CodeValidation)
	case errors.Is(err, entity.ErrDuplicateEmail):
		RespondWithError(c, http.StatusConflict, "このメールアドレスは既に使用されています", CodeConflict)
	case errors.Is(err, entity.ErrDuplicateClerkID):
		RespondWithError(c, http.StatusConflict, "このClerk IDは既に登録されています", CodeConflict)
	case errors.Is(err, entity.ErrUnauthorized):
		RespondWithError(c, http.StatusForbidden, "権限がありません", CodeForbidden)

	case errors.Is(err, entity.ErrFriendRequestAlreadyExists):
		RespondWithError(c, http.StatusConflict, "フレンド申請は既に送信されています", CodeConflict)
	case errors.Is(err, entity.ErrFriendRequestNotFound):
		RespondWithError(c, http.StatusNotFound, "フレンドリクエストが見つかりません", CodeNotFound)
	case errors.Is(err, entity.ErrFriendshipNotFound):
		RespondWithError(c, http.StatusNotFound, "フレンド関係が見つかりません", CodeNotFound)
	case errors.Is(err, entity.ErrAlreadyFriends):
		RespondWithError(c, http.StatusConflict, "すでにフレンドです", CodeConflict)
	case errors.Is(err, entity.ErrCannotSendToSelf):
		RespondWithError(c, http.StatusBadRequest, "自身へのリクエスト送信は行えません", CodeValidation)

	case errors.Is(err, entity.ErrSeatNotFound):
		RespondWithError(c, http.StatusNotFound, "座席が見つかりません", CodeNotFound)
	case errors.Is(err, entity.ErrDuplicateSeatNumber):
		RespondWithError(c, http.StatusConflict, "この座席番号は既に使用されています", CodeConflict)
	case errors.Is(err, entity.ErrInvalidSeatShape):
		RespondWithError(c, http.StatusBadRequest, "無効な座席形状です", CodeValidation)
	case errors.Is(err, entity.ErrInvalidRotation):
		RespondWithError(c, http.StatusBadRequest, "無効な回転角度です（15度刻みで0-359の範囲）", CodeValidation)

	case errors.Is(err, entity.ErrFloorNotFound):
		RespondWithError(c, http.StatusNotFound, "フロアが見つかりません", CodeNotFound)
	case errors.Is(err, entity.ErrDuplicateFloorName):
		RespondWithError(c, http.StatusConflict, "このフロア名は既に使用されています", CodeConflict)

	case errors.Is(err, entity.ErrFloorOperationHoursNotFound):
		RespondWithError(c, http.StatusNotFound, "フロア営業時間が見つかりません", CodeNotFound)
	case errors.Is(err, entity.ErrInvalidDayOfWeek):
		RespondWithError(c, http.StatusBadRequest, "無効な曜日です（0-6の範囲で指定してください）", CodeValidation)
	case errors.Is(err, entity.ErrInvalidOperationHours):
		RespondWithError(c, http.StatusBadRequest, "営業時間が無効です（終了時刻は開始時刻より後である必要があります）", CodeValidation)
	case errors.Is(err, entity.ErrDuplicateFloorDayOfWeek):
		RespondWithError(c, http.StatusConflict, "この曜日の営業時間は既に登録されています", CodeConflict)
	case errors.Is(err, entity.ErrFloorClosed):
		RespondWithError(c, http.StatusBadRequest, "指定された日時はフロアが休館日です", CodeValidation)
	case errors.Is(err, entity.ErrOutsideOperatingHours):
		RespondWithError(c, http.StatusBadRequest, "予約時間が営業時間外です", CodeValidation)

	case errors.Is(err, entity.ErrReservationNotFound):
		RespondWithError(c, http.StatusNotFound, "予約が見つかりません", CodeNotFound)
	case errors.Is(err, entity.ErrReservationOverlap):
		RespondWithError(c, http.StatusConflict, "指定時間帯に既に予約が存在します", CodeConflict)
	case errors.Is(err, entity.ErrInvalidReservationTime):
		RespondWithError(c, http.StatusBadRequest, "無効な予約時間です", CodeValidation)
	case errors.Is(err, entity.ErrCannotCheckIn):
		RespondWithError(c, http.StatusBadRequest, "チェックインできません", CodeBadRequest)
	case errors.Is(err, entity.ErrCannotCheckOut):
		RespondWithError(c, http.StatusBadRequest, "チェックアウトできません", CodeBadRequest)
	case errors.Is(err, entity.ErrCannotCancel):
		RespondWithError(c, http.StatusBadRequest, "キャンセルできません", CodeBadRequest)
	case errors.Is(err, entity.ErrCannotExtend):
		RespondWithError(c, http.StatusBadRequest, "延長できません", CodeBadRequest)
	case errors.Is(err, entity.ErrInvalidReservationType):
		RespondWithError(c, http.StatusBadRequest, "無効な予約タイプです", CodeValidation)
	case errors.Is(err, entity.ErrInvalidReservationStatus):
		RespondWithError(c, http.StatusBadRequest, "無効な予約ステータスです", CodeValidation)

	case errors.Is(err, entity.ErrRecurringReservationNotFound):
		RespondWithError(c, http.StatusNotFound, "定期予約が見つかりません", CodeNotFound)
	case errors.Is(err, entity.ErrInvalidDaysOfWeek):
		RespondWithError(c, http.StatusBadRequest, "無効な曜日指定です", CodeValidation)
	case errors.Is(err, entity.ErrInvalidTimeRange):
		RespondWithError(c, http.StatusBadRequest, "無効な時間範囲です", CodeValidation)

	case errors.Is(err, entity.ErrReservationSettingsNotFound):
		RespondWithError(c, http.StatusNotFound, "予約設定が見つかりません", CodeNotFound)
	case errors.Is(err, entity.ErrReservationTooShort):
		RespondWithError(c, http.StatusBadRequest, "予約時間が最小時間より短いです", CodeValidation)
	case errors.Is(err, entity.ErrReservationTooLong):
		RespondWithError(c, http.StatusBadRequest, "予約時間が最大時間を超えています", CodeValidation)
	case errors.Is(err, entity.ErrAdvanceBookingExceeded):
		RespondWithError(c, http.StatusBadRequest, "予約は事前予約期限内である必要があります", CodeValidation)
	case errors.Is(err, entity.ErrCheckInTooEarly):
		RespondWithError(c, http.StatusBadRequest, "チェックイン可能時間前です", CodeValidation)
	case errors.Is(err, entity.ErrCheckInDeadlinePassed):
		RespondWithError(c, http.StatusBadRequest, "チェックイン期限を過ぎています", CodeValidation)
	case errors.Is(err, entity.ErrExtensionLimitExceeded):
		RespondWithError(c, http.StatusBadRequest, "延長回数の上限に達しています", CodeValidation)
	case errors.Is(err, entity.ErrExtensionTimeTooLong):
		RespondWithError(c, http.StatusBadRequest, "延長時間が長すぎます", CodeValidation)
	case errors.Is(err, entity.ErrCancellationDeadlinePassed):
		RespondWithError(c, http.StatusBadRequest, "キャンセル期限を過ぎています", CodeValidation)
	case errors.Is(err, entity.ErrInstantReservationNotAllowed):
		RespondWithError(c, http.StatusBadRequest, "このインスタント予約は許可されていません", CodeValidation)

	default:
		RespondWithError(c, http.StatusInternalServerError, defaultMessage, CodeInternalError, map[string]interface{}{
			"internal_error": err.Error(),
		})
	}
}

func RespondWithValidationError(c *gin.Context, err error) {
	RespondWithError(c, http.StatusBadRequest, "リクエストが無効です", CodeValidation, map[string]interface{}{
		"validation_error": err.Error(),
	})
}

func RespondWithUnauthorized(c *gin.Context) {
	RespondWithError(c, http.StatusUnauthorized, "認証されていません", CodeUnauthorized)
}
