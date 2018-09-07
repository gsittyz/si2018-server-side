package userlike

import (
	"github.com/go-openapi/runtime/middleware"
	"github.com/go-openapi/strfmt"
	"sort"
	"time"

	"github.com/eure/si2018-server-side/entities"
	"github.com/eure/si2018-server-side/models"
	"github.com/eure/si2018-server-side/repositories"
	si "github.com/eure/si2018-server-side/restapi/summerintern"
)

type UserResponses []*models.LikeUserResponse

func (a UserResponses) Len() int      { return len(a) }
func (a UserResponses) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a UserResponses) Less(i, j int) bool {
	ai := time.Time(a[i].LikedAt)
	aj := time.Time(a[j].LikedAt)
	return !ai.Before(aj)
}

func GetLikes(p si.GetLikesParams) middleware.Responder {
	/*
		1.	tokenのvalidation
		2.	tokenからuseridを取得
		3.	userIDからマッチ済みの相手matchIDを取得
		4.	useridからマッチ済み以外のいいねの受信リストを取得
		5.	いいねの受信リストからユーザーのプロフィールのリストを取得
		userIDはいいねを送った人, partnerIDはいいねを受け取った人
	*/

	// Tokenがあるかどうか
	if p.Token == "" {
		return si.NewGetLikesUnauthorized().WithPayload(
			&si.GetLikesUnauthorizedBody{
				Code:    "401",
				Message: "Token Is Required",
			})
	}

	// tokenからuserIDを取得

	rToken := repositories.NewUserTokenRepository()
	entToken, errToken := rToken.GetByToken(p.Token)
	if errToken != nil {
		return si.NewGetLikesInternalServerError().WithPayload(
			&si.GetLikesInternalServerErrorBody{
				Code:    "500",
				Message: "Internal Server Error",
			})
	}

	if entToken == nil {
		return si.NewGetLikesUnauthorized().WithPayload(
			&si.GetLikesUnauthorizedBody{
				Code:    "401",
				Message: "Token Is Invalid",
			})
	}

	sEntToken := entToken.Build()

	// matchIDsの取得

	rMatch := repositories.NewUserMatchRepository()
	matchIDs, errMatch := rMatch.FindAllByUserID(sEntToken.UserID)

	if errMatch != nil {
		return si.NewGetLikesInternalServerError().WithPayload(
			&si.GetLikesInternalServerErrorBody{
				Code:    "500",
				Message: "Internal Server Error",
			})
	}

	//fmt.Println("matchIDs",matchIDs)
	// マッチ済み以外のいいね受信リストを取得する
	rLike := repositories.NewUserLikeRepository()
	limit := int(p.Limit)
	offset := int(p.Offset)
	if limit <= 0 || offset < 0 {
		return si.NewGetLikesBadRequest().WithPayload(
			&si.GetLikesBadRequestBody{
				"400",
				"Bad Request",
			})
	}
	//fmt.Println("sEntToken.UserID",sEntToken.UserID)
	//fmt.Println("limit",limit)
	//fmt.Println("offset",offset)
	likes, errLike := rLike.FindGotLikeWithLimitOffset(sEntToken.UserID, limit, offset, matchIDs)
	if errLike != nil {
		return si.NewGetLikesInternalServerError().WithPayload(
			&si.GetLikesInternalServerErrorBody{
				Code:    "500",
				Message: "Internal Server Error",
			})
	}

	// fmt.Println("likes",likes)
	userLikes := entities.UserLikes(likes)

	sUsers := userLikes.Build() // userID(送信元) partnerID（送信先） createdAt UpdatedAtのリスト

	// id -- 時間の対応mapとpartneridのリストの作成
	partnerLikedAt := map[int64]strfmt.DateTime{}
	var IDs []int64
	for _, sUser := range sUsers {
		partnerLikedAt[sUser.UserID] = sUser.CreatedAt
		IDs = append(IDs, sUser.UserID)
	}

	rUser := repositories.NewUserRepository()

	// 上で取得した全てのpartnerIDについて、プロフィール情報と画像URIを取得してpayloadsに格納する。
	partners, errFind := rUser.FindByIDs(IDs)
	// fmt.Println("partners",partners)
	if errFind != nil {
		return si.NewGetLikesInternalServerError().WithPayload(
			&si.GetLikesInternalServerErrorBody{
				Code:    "500",
				Message: "Internal Server Error",
			})
	}

	// 画像URI取得
	rImage := repositories.NewUserImageRepository()
	entImages, errImages := rImage.GetByUserIDs(IDs)
	if errImages != nil || entImages == nil {
		return si.NewGetProfileByUserIDInternalServerError().WithPayload(
			&si.GetProfileByUserIDInternalServerErrorBody{
				Code:    "500",
				Message: "Internal Server Error",
			})
	}

	// id -- pathの対応リストを作成
	idPaths := map[int64]string{}
	for _, entImage := range entImages {
		idPaths[entImage.UserID] = entImage.Path
	}

	var payloads []*models.LikeUserResponse
	for _, partner := range partners {
		var r entities.LikeUserResponse
		r.ApplyUser(partner)
		r.LikedAt = partnerLikedAt[partner.ID]
		r.ImageURI = idPaths[partner.ID]
		m := r.Build()
		payloads = append(payloads, &m)
	}

	sort.Sort(UserResponses(payloads))

	//fmt.Println("payloads",payloads)
	return si.NewGetLikesOK().WithPayload(payloads)
}

func PostLike(p si.PostLikeParams) middleware.Responder {
	/*
		1.	Tokenのバリデーション
		2.	tokenから送信者のuseridを取得
		3.	送信者のuseridから送信者のプロフィルを持ってきて性別を確認
		4.	p.useridから送信相手のプロフィルを持ってきて異性かどうか確認
		5.	すでにいいねしているか確認
		6.	いいねを送信
	*/

	// Tokenがあるかどうか
	if p.Params.Token == "" {
		return si.NewPostLikeUnauthorized().WithPayload(
			&si.PostLikeUnauthorizedBody{
				Code:    "401",
				Message: "Token Is Required",
			})
	}

	// tokenから送信者のuserIDを取得

	rToken := repositories.NewUserTokenRepository()
	entToken, errToken := rToken.GetByToken(p.Params.Token)
	if errToken != nil {
		return si.NewPostLikeInternalServerError().WithPayload(
			&si.PostLikeInternalServerErrorBody{
				Code:    "500",
				Message: "Internal Server Error",
			})
	}

	if entToken == nil {
		return si.NewPostLikeUnauthorized().WithPayload(
			&si.PostLikeUnauthorizedBody{
				Code:    "401",
				Message: "Token Is Invalid",
			})
	}

	sEntToken := entToken.Build()

	// 送信者のuseridから送信者のプロフィルを持ってきて性別を確認
	// genderを確認するためだけに、useridからプロフィルを取得する……
	rUser := repositories.NewUserRepository()
	entUser, errUser := rUser.GetByUserID(sEntToken.UserID)
	if errUser != nil {
		return si.NewPostLikeInternalServerError().WithPayload(
			&si.PostLikeInternalServerErrorBody{
				Code:    "500",
				Message: "Internal Server Error",
			})
	}

	if entUser == nil { // entUserがnilになることはないはずだが、一応書いておく
		return si.NewPostLikeInternalServerError().WithPayload(
			&si.PostLikeInternalServerErrorBody{
				Code:    "500",
				Message: "Internal Server Error",
			})
	}

	gender := entUser.GetOppositeGender()

	// 送信相手のuseridから送信相手のプロフィルを持ってきて性別を確認
	// genderを確認するためだけに、useridからプロフィルを取得する……

	// userを設定する
	entUser2, errUser2 := rUser.GetByUserID(p.UserID)
	if errUser2 != nil {
		return si.NewPostLikeInternalServerError().WithPayload(
			&si.PostLikeInternalServerErrorBody{
				Code:    "500",
				Message: "Internal Server Error",
			})
	}

	if entUser2 == nil { // 存在しない送信相手を指定した場合
		return si.NewPostLikeBadRequest().WithPayload(
			&si.PostLikeBadRequestBody{
				Code:    "400",
				Message: "Bad Request",
			})
	}

	// 異性かどうかの確認
	if entUser2.Gender != gender {
		return si.NewPostLikeBadRequest().WithPayload(
			&si.PostLikeBadRequestBody{
				Code:    "400",
				Message: "Bad Request",
			})
	}

	// すでにいいねしているかどうか確認する
	// userIDはいいねを送った人, partnerIDはいいねを受け取った人
	rLike := repositories.NewUserLikeRepository()
	entLike, errLike := rLike.GetLikeBySenderIDReceiverID(sEntToken.UserID, p.UserID)
	if errLike != nil {
		return si.NewPostLikeInternalServerError().WithPayload(
			&si.PostLikeInternalServerErrorBody{
				Code:    "500",
				Message: "Internal Server Error",
			})
	}
	// すでにいいねしている場合
	if entLike != nil {
		return si.NewPostLikeBadRequest().WithPayload(
			&si.PostLikeBadRequestBody{
				Code:    "400",
				Message: "Already Liked",
			})
	}

	var userLike entities.UserLike
	userLike.UserID = sEntToken.UserID
	userLike.PartnerID = p.UserID
	userLike.CreatedAt = strfmt.DateTime(time.Now())
	userLike.UpdatedAt = userLike.CreatedAt
	// いいねを送信する
	errLikeCreate := rLike.Create(userLike)
	if errLikeCreate != nil {
		//fmt.Println(errLikeCreate)
		return si.NewPostLikeInternalServerError().WithPayload(
			&si.PostLikeInternalServerErrorBody{
				Code:    "500",
				Message: "Internal Server Error",
			})
	}

	return si.NewPostLikeOK().WithPayload(
		&si.PostLikeOKBody{
			Code:    "200",
			Message: "OK",
		})
}
