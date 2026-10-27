package controller

import (
	"context"
	"log"
	"net/http"

	"github.com/CMS-Enterprise/ztmf/backend/internal/model"
	"github.com/gorilla/mux"
)

// Questionnaire version lifecycle (ztmf-misc#396). Every write here changes
// what every OpDiv is scored on, so all of them sit behind the same
// CanWriteHHSWide gate as the catalog content routes (ztmf-misc#398), checked
// before the body is read or the database touched.

// @Summary	List questionnaire versions
// @Tags		questionnaire-versions
// @Produce	json
// @Security	bearerAuth
// @Success	200	{object}	apiResponse[[]model.QuestionnaireVersion]
// @Failure	500	{object}	apiResponse[any]
// @Router		/questionnaire-versions [get]
func ListQuestionnaireVersions(w http.ResponseWriter, r *http.Request) {
	versions, err := model.FindQuestionnaireVersions(r.Context())
	respond(w, r, versions, err)
}

// @Summary	Get a questionnaire version
// @Tags		questionnaire-versions
// @Produce	json
// @Security	bearerAuth
// @Param		versionid	path		int	true	"Questionnaire version ID"
// @Success	200			{object}	apiResponse[model.QuestionnaireVersion]
// @Failure	404			{object}	apiResponse[any]
// @Failure	500			{object}	apiResponse[any]
// @Router		/questionnaire-versions/{versionid} [get]
func GetQuestionnaireVersion(w http.ResponseWriter, r *http.Request) {
	id, ok := versionIDFromPath(r)
	if !ok {
		respond(w, r, nil, ErrNotFound)
		return
	}
	version, err := model.FindQuestionnaireVersionByID(r.Context(), id)
	respond(w, r, version, err)
}

type forkQuestionnaireVersionInput struct {
	Label string `json:"label"`
}

// ForkQuestionnaireVersion opens a draft copied from the current published
// version. A second draft is a 400.
//
//	@Summary	Fork a draft from the current published questionnaire version
//	@Tags		questionnaire-versions
//	@Accept		json
//	@Produce	json
//	@Security	bearerAuth
//	@Param		body	body		forkQuestionnaireVersionInput	true	"Label for the draft"
//	@Success	201		{object}	apiResponse[model.QuestionnaireVersion]
//	@Failure	400		{object}	apiResponse[any]
//	@Failure	403		{object}	apiResponse[any]
//	@Failure	500		{object}	apiResponse[any]
//	@Router		/questionnaire-versions [post]
func ForkQuestionnaireVersion(w http.ResponseWriter, r *http.Request) {
	if !model.UserFromContext(r.Context()).CanWriteHHSWide() {
		respond(w, r, nil, ErrForbidden)
		return
	}
	input := forkQuestionnaireVersionInput{}
	if err := getJSON(r.Body, &input); err != nil {
		log.Println(err)
		respond(w, r, nil, ErrMalformed)
		return
	}
	version, err := model.ForkQuestionnaireVersion(r.Context(), input.Label)
	respond(w, r, version, err)
}

// @Summary	Publish a draft questionnaire version
// @Description	Runs the publish gate; a 400 names the offending rows for each rule that fails.
// @Tags		questionnaire-versions
// @Produce	json
// @Security	bearerAuth
// @Param		versionid	path		int	true	"Questionnaire version ID"
// @Success	200			{object}	apiResponse[model.QuestionnaireVersion]
// @Failure	400			{object}	apiResponse[any]
// @Failure	403			{object}	apiResponse[any]
// @Failure	404			{object}	apiResponse[any]
// @Failure	500			{object}	apiResponse[any]
// @Router		/questionnaire-versions/{versionid}/publish [put]
func PublishQuestionnaireVersion(w http.ResponseWriter, r *http.Request) {
	versionTransition(w, r, model.PublishQuestionnaireVersion)
}

// @Summary	Archive a published questionnaire version
// @Description	Refused for the current published version. Data calls pinned to an archived version keep resolving it.
// @Tags		questionnaire-versions
// @Produce	json
// @Security	bearerAuth
// @Param		versionid	path		int	true	"Questionnaire version ID"
// @Success	200			{object}	apiResponse[model.QuestionnaireVersion]
// @Failure	400			{object}	apiResponse[any]
// @Failure	403			{object}	apiResponse[any]
// @Failure	404			{object}	apiResponse[any]
// @Failure	500			{object}	apiResponse[any]
// @Router		/questionnaire-versions/{versionid}/archive [put]
func ArchiveQuestionnaireVersion(w http.ResponseWriter, r *http.Request) {
	versionTransition(w, r, model.ArchiveQuestionnaireVersion)
}

// @Summary	Discard a draft questionnaire version
// @Tags		questionnaire-versions
// @Security	bearerAuth
// @Param		versionid	path	int	true	"Questionnaire version ID"
// @Success	204			"No Content"
// @Failure	400			{object}	apiResponse[any]
// @Failure	403			{object}	apiResponse[any]
// @Failure	404			{object}	apiResponse[any]
// @Failure	500			{object}	apiResponse[any]
// @Router		/questionnaire-versions/{versionid} [delete]
func DiscardQuestionnaireVersion(w http.ResponseWriter, r *http.Request) {
	if !model.UserFromContext(r.Context()).CanWriteHHSWide() {
		respond(w, r, nil, ErrForbidden)
		return
	}
	id, ok := versionIDFromPath(r)
	if !ok {
		respond(w, r, nil, ErrNotFound)
		return
	}
	respond(w, r, nil, model.DiscardQuestionnaireVersion(r.Context(), id))
}

// versionTransition is the shared shape of publish and archive: gate, path id,
// transition, and the updated row back with a 200 (house style, as
// /users/{id}/restore).
func versionTransition(w http.ResponseWriter, r *http.Request, transition func(ctx context.Context, id int32) (*model.QuestionnaireVersion, error)) {
	if !model.UserFromContext(r.Context()).CanWriteHHSWide() {
		respond(w, r, nil, ErrForbidden)
		return
	}
	id, ok := versionIDFromPath(r)
	if !ok {
		respond(w, r, nil, ErrNotFound)
		return
	}
	version, err := transition(r.Context(), id)
	if err != nil {
		respond(w, r, nil, err)
		return
	}
	respondOK(w, version)
}

func versionIDFromPath(r *http.Request) (int32, bool) {
	return pathInt32(mux.Vars(r)["versionid"])
}
