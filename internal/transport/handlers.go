package transport

import (
	"fmt"
	"io"
	"net/http"
	"time"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/gertd/go-pluralize"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/nasermirzaei89/core/internal/core"
	"github.com/nasermirzaei89/core/internal/repository"
	"github.com/nasermirzaei89/core/lib/problem"
	"github.com/nasermirzaei89/respond"
	"github.com/pkg/errors"
)

func (h *Handler) CreateItemHandler() http.HandlerFunc {
	type Response struct {
		respond.WithStatusCreated
		core.Item
	}

	return func(w http.ResponseWriter, r *http.Request) {
		pc := pluralize.NewClient()

		typePlural := mux.Vars(r)["typePlural"]

		if !pc.IsPlural(typePlural) {
			respond.Done(w, r, problem.BadRequest("you should set plural form of the type"))

			return
		}

		typ := pc.Singular(typePlural)

		if !isValidType(typ) {
			respond.Done(w, r, problem.BadRequest(fmt.Sprintf("type field is not valid, it should an string that matches the regex '%s'", core.TypeRegex)))

			return
		}

		var req core.Item

		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			respond.Done(w, r, problem.BadRequest("error on decode request body", problem.WithExtension("error", err.Error())))

			return
		}

		if req.Name == "" {
			respond.Done(w, r, problem.BadRequest("name field is required"))

			return
		}

		if !isValidName(req.Name) {
			respond.Done(w, r, problem.BadRequest(fmt.Sprintf("name field is not valid, it should an string that matches the regex '%s'", core.NameRegex)))

			return
		}

		_, err = h.itemRepo.GetByTypeAndName(r.Context(), typ, req.Name)
		if err != nil {
			if !errors.Is(err, repository.ErrItemNotFound) {
				respond.Done(w, r, problem.InternalServerError(errors.Wrap(err, "error on find item by type and name from the repository")))

				return
			}
		} else {
			respond.Done(w, r, problem.Conflict(fmt.Sprintf("%s with name '%s' already exists", typ, req.Name)))

			return
		}

		now := time.Now()

		item := core.Item{
			UUID:      uuid.NewString(),
			Type:      typ,
			Name:      req.Name,
			Data:      req.Data,
			CreatedAt: now,
			UpdatedAt: now,
		}

		err = h.itemRepo.Insert(r.Context(), item)
		if err != nil {
			respond.Done(w, r, problem.InternalServerError(errors.Wrap(err, "error on insert item to the repository")))

			return
		}

		rsp := Response{Item: item}

		respond.Done(w, r, rsp)
	}
}

func (h *Handler) ListItemsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pc := pluralize.NewClient()

		typePlural := mux.Vars(r)["typePlural"]

		if !pc.IsPlural(typePlural) {
			respond.Done(w, r, problem.BadRequest("you should set plural form of the type"))

			return
		}

		typ := pc.Singular(typePlural)

		if !isValidType(typ) {
			respond.Done(w, r, problem.BadRequest(fmt.Sprintf("type field is not valid, it should an string that matches the regex '%s'", core.TypeRegex)))

			return
		}

		items, err := h.itemRepo.ListByType(r.Context(), typ)
		if err != nil {
			respond.Done(w, r, problem.InternalServerError(errors.Wrap(err, "error on list items by type from the repository")))

			return
		}

		rsp := core.ItemList{Items: items}

		respond.Done(w, r, rsp)
	}
}

func (h *Handler) ReadItemHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pc := pluralize.NewClient()

		typePlural := mux.Vars(r)["typePlural"]

		if !pc.IsPlural(typePlural) {
			respond.Done(w, r, problem.BadRequest("you should set plural form of the type"))

			return
		}

		typ := pc.Singular(typePlural)

		if !isValidType(typ) {
			respond.Done(w, r, problem.BadRequest(fmt.Sprintf("type parameter is not valid, it should an string that matches the regex '%s'", core.TypeRegex)))

			return
		}

		name := mux.Vars(r)["name"]

		if !isValidName(name) {
			respond.Done(w, r, problem.BadRequest(fmt.Sprintf("name parameter is not valid, it should an string that matches the regex '%s'", core.NameRegex)))

			return
		}

		item, err := h.itemRepo.GetByTypeAndName(r.Context(), typ, name)
		if err != nil {
			switch {
			case errors.Is(err, repository.ErrItemNotFound):
				respond.Done(w, r, problem.NotFound(fmt.Sprintf("%s with name '%s' not found", typ, name)))
			default:
				respond.Done(w, r, problem.InternalServerError(errors.Wrap(err, "error on find item by type and name from the repository")))
			}

			return
		}

		respond.Done(w, r, item)
	}
}

func (h *Handler) ReplaceItemHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pc := pluralize.NewClient()

		typePlural := mux.Vars(r)["typePlural"]

		if !pc.IsPlural(typePlural) {
			respond.Done(w, r, problem.BadRequest("you should set plural form of the type"))

			return
		}

		typ := pc.Singular(typePlural)

		if !isValidType(typ) {
			respond.Done(w, r, problem.BadRequest(fmt.Sprintf("type parameter is not valid, it should an string that matches the regex '%s'", core.TypeRegex)))

			return
		}

		name := mux.Vars(r)["name"]

		if !isValidName(name) {
			respond.Done(w, r, problem.BadRequest(fmt.Sprintf("name parameter is not valid, it should an string that matches the regex '%s'", core.NameRegex)))

			return
		}

		var req core.Item

		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			respond.Done(w, r, problem.BadRequest("error on decode request body", problem.WithExtension("error", err.Error())))

			return
		}

		item, err := h.itemRepo.GetByTypeAndName(r.Context(), typ, name)
		if err != nil {
			if errors.Is(err, repository.ErrItemNotFound) {
				respond.Done(w, r, problem.NotFound(fmt.Sprintf("%s with name '%s' not found", typ, name)))
			} else {
				respond.Done(w, r, problem.InternalServerError(errors.Wrap(err, "error on find item by type and name from the repository")))
			}

			return
		}

		item.Data = req.Data
		item.UpdatedAt = time.Now()

		err = h.itemRepo.Replace(r.Context(), item.UUID, *item)
		if err != nil {
			respond.Done(w, r, problem.InternalServerError(errors.Wrap(err, "error on replace item in the repository")))

			return
		}

		respond.Done(w, r, *item)
	}
}

func (h *Handler) PatchItemHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pc := pluralize.NewClient()

		typePlural := mux.Vars(r)["typePlural"]

		if !pc.IsPlural(typePlural) {
			respond.Done(w, r, problem.BadRequest("you should set plural form of the type"))

			return
		}

		typ := pc.Singular(typePlural)

		if !isValidType(typ) {
			respond.Done(w, r, problem.BadRequest(fmt.Sprintf("type parameter is not valid, it should an string that matches the regex '%s'", core.TypeRegex)))

			return
		}

		name := mux.Vars(r)["name"]

		if !isValidName(name) {
			respond.Done(w, r, problem.BadRequest(fmt.Sprintf("name parameter is not valid, it should an string that matches the regex '%s'", core.NameRegex)))

			return
		}

		item, err := h.itemRepo.GetByTypeAndName(r.Context(), typ, name)
		if err != nil {
			if errors.Is(err, repository.ErrItemNotFound) {
				respond.Done(w, r, problem.NotFound(fmt.Sprintf("%s with name '%s' not found", typ, name)))
			} else {
				respond.Done(w, r, problem.InternalServerError(errors.Wrap(err, "error on find item by type and name from the repository")))
			}

			return
		}

		originalBytes, err := json.Marshal(item)
		if err != nil {
			respond.Done(w, r, problem.InternalServerError(errors.Wrap(err, "error on marshal original item")))

			return
		}

		requestBody, err := io.ReadAll(r.Body)
		if err != nil {
			respond.Done(w, r, problem.InternalServerError(errors.Wrap(err, "error on read request body")))

			return
		}

		ctype := r.Header.Get("Content-Type")

		var modifiedBytes []byte

		switch ctype {
		case "application/json-patch+json":
			patch, err := jsonpatch.DecodePatch(requestBody)
			if err != nil {
				respond.Done(w, r, problem.BadRequest("error on decode json patch", problem.WithExtension("error", err.Error())))

				return
			}

			modifiedBytes, err = patch.Apply(originalBytes)
			if err != nil {
				respond.Done(w, r, problem.InternalServerError(errors.Wrap(err, "error on apply json patch")))

				return
			}
		case "application/merge-patch+json":
			modifiedBytes, err = jsonpatch.MergePatch(originalBytes, requestBody)
			if err != nil {
				respond.Done(w, r, problem.BadRequest("error on apply merge patch", problem.WithExtension("error", err.Error())))

				return
			}
		default:
			respond.Done(w, r, problem.BadRequest("unsupported Content-Type header"))

			return
		}

		var modified core.Item

		err = json.Unmarshal(modifiedBytes, &modified)
		if err != nil {
			respond.Done(w, r, problem.BadRequest("error on unmarshal modified bytes", problem.WithExtension("error", err.Error())))

			return
		}

		item.Data = modified.Data
		item.UpdatedAt = time.Now()

		err = h.itemRepo.Replace(r.Context(), item.UUID, *item)
		if err != nil {
			respond.Done(w, r, problem.InternalServerError(errors.Wrap(err, "error on replace item in the repository")))

			return
		}

		respond.Done(w, r, *item)
	}
}

func (h *Handler) DeleteItemHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pc := pluralize.NewClient()

		typePlural := mux.Vars(r)["typePlural"]

		if !pc.IsPlural(typePlural) {
			respond.Done(w, r, problem.BadRequest("you should set plural form of the type"))

			return
		}

		typ := pc.Singular(typePlural)

		if !isValidType(typ) {
			respond.Done(w, r, problem.BadRequest(fmt.Sprintf("type parameter is not valid, it should an string that matches the regex '%s'", core.TypeRegex)))

			return
		}

		name := mux.Vars(r)["name"]

		if !isValidName(name) {
			respond.Done(w, r, problem.BadRequest(fmt.Sprintf("name parameter is not valid, it should an string that matches the regex '%s'", core.NameRegex)))

			return
		}

		item, err := h.itemRepo.GetByTypeAndName(r.Context(), typ, name)
		if err != nil {
			if errors.Is(err, repository.ErrItemNotFound) {
				respond.Done(w, r, problem.NotFound(fmt.Sprintf("%s with name '%s' not found", typ, name)))
			} else {
				respond.Done(w, r, problem.InternalServerError(errors.Wrap(err, "error on find item by type and name from the repository")))
			}

			return
		}

		err = h.itemRepo.Delete(r.Context(), item.UUID)
		if err != nil {
			respond.Done(w, r, problem.InternalServerError(errors.Wrap(err, "error on delete item from the repository")))

			return
		}

		respond.Done(w, r, nil)
	}
}
