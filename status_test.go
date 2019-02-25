package httpc_test

import (
	"net/http"
	"testing"

	"github.com/jsteenb2/httpc"
)

func TestStatusFuncs(t *testing.T) {
	t.Run("single match", func(t *testing.T) {
		tests := []struct {
			statusFn   httpc.StatusFn
			statusCode int
		}{
			{
				statusCode: http.StatusOK,
				statusFn:   httpc.StatusOK(),
			},
			{
				statusCode: http.StatusNoContent,
				statusFn:   httpc.StatusNoContent(),
			},
			{
				statusCode: http.StatusNotFound,
				statusFn:   httpc.StatusNotFound(),
			},
			{
				statusCode: http.StatusUnprocessableEntity,
				statusFn:   httpc.StatusUnprocessableEntity(),
			},
			{
				statusCode: http.StatusInternalServerError,
				statusFn:   httpc.StatusInternalServerError(),
			},
		}

		for _, tt := range tests {
			fn := func(t *testing.T) {
				equals(t, true, tt.statusFn(tt.statusCode))
			}

			t.Run(http.StatusText(tt.statusCode), fn)
		}
	})

	t.Run("in matches", func(t *testing.T) {
		tests := []struct {
			name  string
			input []int
		}{
			{
				name:  "200s",
				input: []int{http.StatusOK, http.StatusCreated, http.StatusAccepted, http.StatusNoContent},
			},
			{
				name:  "300s",
				input: []int{http.StatusTemporaryRedirect, http.StatusMovedPermanently},
			},
			{
				name:  "400s",
				input: []int{http.StatusNotFound, http.StatusForbidden, http.StatusUnprocessableEntity},
			},
			{
				name:  "500s",
				input: []int{http.StatusInternalServerError, http.StatusBadGateway},
			},
		}

		for _, tt := range tests {
			fn := func(t *testing.T) {
				for _, testcase := range tt.input {
					equals(t, true, httpc.StatusIn(tt.input...)(testcase))
				}
			}
			t.Run(tt.name, fn)
		}
	})

	t.Run("not in matches", func(t *testing.T) {
		tests := []struct {
			name  string
			input []int
		}{
			{
				name:  "200s",
				input: []int{http.StatusOK, http.StatusCreated, http.StatusAccepted, http.StatusNoContent},
			},
			{
				name:  "300s",
				input: []int{http.StatusTemporaryRedirect, http.StatusMovedPermanently},
			},
			{
				name:  "400s",
				input: []int{http.StatusNotFound, http.StatusForbidden, http.StatusUnprocessableEntity},
			},
			{
				name:  "500s",
				input: []int{http.StatusInternalServerError, http.StatusBadGateway},
			},
		}

		for _, tt := range tests {
			fn := func(t *testing.T) {
				for _, testcase := range tt.input {
					equals(t, true, httpc.StatusNotIn(tt.input...)(testcase+100))
				}
			}
			t.Run(tt.name, fn)
		}
	})
}
