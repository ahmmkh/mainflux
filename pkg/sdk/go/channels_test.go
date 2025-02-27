// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package sdk_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-zoo/bone"
	"github.com/mainflux/mainflux/internal/apiutil"
	"github.com/mainflux/mainflux/internal/testsutil"
	"github.com/mainflux/mainflux/logger"
	mfclients "github.com/mainflux/mainflux/pkg/clients"
	"github.com/mainflux/mainflux/pkg/errors"
	mfgroups "github.com/mainflux/mainflux/pkg/groups"
	sdk "github.com/mainflux/mainflux/pkg/sdk/go"
	"github.com/mainflux/mainflux/things/clients"
	"github.com/mainflux/mainflux/things/clients/mocks"
	"github.com/mainflux/mainflux/things/groups"
	"github.com/mainflux/mainflux/things/groups/api"
	gmocks "github.com/mainflux/mainflux/things/groups/mocks"
	"github.com/mainflux/mainflux/things/policies"
	papi "github.com/mainflux/mainflux/things/policies/api/http"
	pmocks "github.com/mainflux/mainflux/things/policies/mocks"
	umocks "github.com/mainflux/mainflux/users/clients/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newChannelsServer(csvc clients.Service, svc groups.Service, psvc policies.Service) *httptest.Server {
	logger := logger.NewMock()
	mux := bone.New()
	api.MakeHandler(svc, mux, logger)
	papi.MakeHandler(csvc, psvc, mux, logger)
	return httptest.NewServer(mux)
}

func TestCreateChannel(t *testing.T) {
	cRepo := new(mocks.Repository)
	gRepo := new(gmocks.Repository)
	pRepo := new(pmocks.Repository)
	uauth := umocks.NewAuthService(users, map[string][]umocks.SubjectSet{adminID: {uadminPolicy}})
	thingCache := mocks.NewCache()
	policiesCache := pmocks.NewCache()

	psvc := policies.NewService(uauth, pRepo, policiesCache, idProvider)

	csvc := clients.NewService(uauth, psvc, cRepo, gRepo, thingCache, idProvider)
	svc := groups.NewService(uauth, psvc, gRepo, idProvider)

	ts := newChannelsServer(csvc, svc, psvc)
	defer ts.Close()

	channel := sdk.Channel{
		Name:     "channelName",
		Metadata: validMetadata,
		Status:   mfclients.EnabledStatus.String(),
	}

	conf := sdk.Config{
		ThingsURL: ts.URL,
	}
	mfsdk := sdk.NewSDK(conf)
	cases := []struct {
		desc    string
		channel sdk.Channel
		token   string
		err     errors.SDKError
	}{
		{
			desc:    "create channel successfully",
			channel: channel,
			token:   token,
			err:     nil,
		},
		{
			desc:    "create channel with existing name",
			channel: channel,
			err:     nil,
		},
		{
			desc: "update channel that can't be marshalled",
			channel: sdk.Channel{
				Name: "test",
				Metadata: map[string]interface{}{
					"test": make(chan int),
				},
			},
			token: token,
			err:   errors.NewSDKError(fmt.Errorf("json: unsupported type: chan int")),
		},
		{
			desc: "create channel with parent",
			channel: sdk.Channel{
				Name:     gName,
				ParentID: testsutil.GenerateUUID(t, idProvider),
				Status:   mfclients.EnabledStatus.String(),
			},
			err: nil,
		},
		{
			desc: "create channel with invalid parent",
			channel: sdk.Channel{
				Name:     gName,
				ParentID: gmocks.WrongID,
				Status:   mfclients.EnabledStatus.String(),
			},
			err: errors.NewSDKErrorWithStatus(errors.ErrCreateEntity, http.StatusInternalServerError),
		},
		{
			desc: "create channel with invalid owner",
			channel: sdk.Channel{
				Name:    gName,
				OwnerID: gmocks.WrongID,
				Status:  mfclients.EnabledStatus.String(),
			},
			err: errors.NewSDKErrorWithStatus(sdk.ErrFailedCreation, http.StatusInternalServerError),
		},
		{
			desc: "create channel with missing name",
			channel: sdk.Channel{
				Status: mfclients.EnabledStatus.String(),
			},
			err: errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrNameSize), http.StatusBadRequest),
		},
		{
			desc: "create a channel with every field defined",
			channel: sdk.Channel{
				ID:          generateUUID(t),
				OwnerID:     "owner",
				ParentID:    "parent",
				Name:        "name",
				Description: description,
				Metadata:    validMetadata,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
				Status:      mfclients.EnabledStatus.String(),
			},
			token: token,
			err:   nil,
		},
	}
	for _, tc := range cases {
		repoCall := gRepo.On("Save", mock.Anything, mock.Anything).Return(convertChannel(sdk.Channel{}), tc.err)
		rChannel, err := mfsdk.CreateChannel(tc.channel, adminToken)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		if err == nil {
			assert.NotEmpty(t, rChannel, fmt.Sprintf("%s: expected not nil on client ID", tc.desc))
			ok := repoCall.Parent.AssertCalled(t, "Save", mock.Anything, mock.Anything)
			assert.True(t, ok, fmt.Sprintf("Save was not called on %s", tc.desc))
		}
		repoCall.Unset()
	}
}

func TestCreateChannels(t *testing.T) {
	cRepo := new(mocks.Repository)
	gRepo := new(gmocks.Repository)
	pRepo := new(pmocks.Repository)
	uauth := umocks.NewAuthService(users, map[string][]umocks.SubjectSet{adminID: {uadminPolicy}})
	thingCache := mocks.NewCache()
	policiesCache := pmocks.NewCache()

	psvc := policies.NewService(uauth, pRepo, policiesCache, idProvider)

	csvc := clients.NewService(uauth, psvc, cRepo, gRepo, thingCache, idProvider)
	svc := groups.NewService(uauth, psvc, gRepo, idProvider)

	ts := newChannelsServer(csvc, svc, psvc)
	defer ts.Close()

	channels := []sdk.Channel{
		{
			Name:     "channelName",
			Metadata: validMetadata,
			Status:   mfclients.EnabledStatus.String(),
		},
		{
			Name:     "channelName2",
			Metadata: validMetadata,
			Status:   mfclients.EnabledStatus.String(),
		},
	}

	conf := sdk.Config{
		ThingsURL: ts.URL,
	}
	mfsdk := sdk.NewSDK(conf)
	cases := []struct {
		desc     string
		channels []sdk.Channel
		response []sdk.Channel
		token    string
		err      errors.SDKError
	}{
		{
			desc:     "create channels successfully",
			channels: channels,
			response: channels,
			token:    token,
			err:      nil,
		},
		{
			desc:     "register empty channels",
			channels: []sdk.Channel{},
			response: []sdk.Channel{},
			token:    token,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrEmptyList), http.StatusBadRequest),
		},
		{
			desc: "register channels that can't be marshalled",
			channels: []sdk.Channel{
				{
					Name: "test",
					Metadata: map[string]interface{}{
						"test": make(chan int),
					},
				},
			},
			response: []sdk.Channel{},
			token:    token,
			err:      errors.NewSDKError(fmt.Errorf("json: unsupported type: chan int")),
		},
	}
	for _, tc := range cases {
		repoCall := gRepo.On("Save", mock.Anything, mock.Anything).Return(convertChannels(tc.response), tc.err)
		rChannel, err := mfsdk.CreateChannels(tc.channels, adminToken)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		if err == nil {
			assert.NotEmpty(t, rChannel, fmt.Sprintf("%s: expected not nil on client ID", tc.desc))
			ok := repoCall.Parent.AssertCalled(t, "Save", mock.Anything, mock.Anything)
			assert.True(t, ok, fmt.Sprintf("Save was not called on %s", tc.desc))
		}
		repoCall.Unset()
	}
}

func TestListChannels(t *testing.T) {
	cRepo := new(mocks.Repository)
	gRepo := new(gmocks.Repository)
	pRepo := new(pmocks.Repository)
	uauth := umocks.NewAuthService(users, map[string][]umocks.SubjectSet{adminID: {uadminPolicy}})
	thingCache := mocks.NewCache()
	policiesCache := pmocks.NewCache()

	psvc := policies.NewService(uauth, pRepo, policiesCache, idProvider)

	csvc := clients.NewService(uauth, psvc, cRepo, gRepo, thingCache, idProvider)
	svc := groups.NewService(uauth, psvc, gRepo, idProvider)

	ts := newChannelsServer(csvc, svc, psvc)
	defer ts.Close()

	var chs []sdk.Channel
	conf := sdk.Config{
		ThingsURL: ts.URL,
	}
	mfsdk := sdk.NewSDK(conf)

	for i := 10; i < 100; i++ {
		gr := sdk.Channel{
			ID:       generateUUID(t),
			Name:     fmt.Sprintf("channel_%d", i),
			Metadata: sdk.Metadata{"name": fmt.Sprintf("thing_%d", i)},
			Status:   mfclients.EnabledStatus.String(),
		}
		chs = append(chs, gr)
	}

	cases := []struct {
		desc     string
		token    string
		status   mfclients.Status
		total    uint64
		offset   uint64
		limit    uint64
		level    int
		name     string
		ownerID  string
		metadata sdk.Metadata
		err      errors.SDKError
		response []sdk.Channel
	}{
		{
			desc:     "get a list of channels",
			token:    token,
			limit:    limit,
			offset:   offset,
			total:    total,
			err:      nil,
			response: chs[offset:limit],
		},
		{
			desc:     "get a list of channels with invalid token",
			token:    invalidToken,
			offset:   offset,
			limit:    limit,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, sdk.ErrFailedList), http.StatusInternalServerError),
			response: nil,
		},
		{
			desc:     "get a list of channels with empty token",
			token:    "",
			offset:   offset,
			limit:    limit,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, sdk.ErrFailedList), http.StatusInternalServerError),
			response: nil,
		},
		{
			desc:     "get a list of channels with zero limit",
			token:    token,
			offset:   offset,
			limit:    0,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, sdk.ErrFailedList), http.StatusInternalServerError),
			response: nil,
		},
		{
			desc:     "get a list of channels with limit greater than max",
			token:    token,
			offset:   offset,
			limit:    110,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, sdk.ErrFailedList), http.StatusInternalServerError),
			response: []sdk.Channel(nil),
		},
		{
			desc:     "get a list of channels with given name",
			token:    token,
			offset:   0,
			limit:    1,
			err:      nil,
			metadata: sdk.Metadata{},
			response: []sdk.Channel{chs[89]},
		},
		{
			desc:     "get a list of channels with level",
			token:    token,
			offset:   0,
			limit:    1,
			level:    1,
			err:      nil,
			response: []sdk.Channel{chs[0]},
		},
		{
			desc:     "get a list of channels with metadata",
			token:    token,
			offset:   0,
			limit:    1,
			err:      nil,
			metadata: sdk.Metadata{},
			response: []sdk.Channel{chs[89]},
		},
	}

	for _, tc := range cases {
		repoCall := pRepo.On("EvaluateGroupAccess", mock.Anything, mock.Anything).Return(policies.Policy{}, nil)
		repoCall1 := gRepo.On("RetrieveAll", mock.Anything, mock.Anything).Return(mfgroups.GroupsPage{Groups: convertChannels(tc.response)}, tc.err)
		pm := sdk.PageMetadata{}
		page, err := mfsdk.Channels(pm, adminToken)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
		assert.Equal(t, len(tc.response), len(page.Channels), fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, page))
		if tc.err == nil {
			ok := repoCall1.Parent.AssertCalled(t, "RetrieveAll", mock.Anything, mock.Anything)
			assert.True(t, ok, fmt.Sprintf("RetrieveAll was not called on %s", tc.desc))
		}
		repoCall.Unset()
		repoCall1.Unset()
	}
}

func TestViewChannel(t *testing.T) {
	cRepo := new(mocks.Repository)
	gRepo := new(gmocks.Repository)
	pRepo := new(pmocks.Repository)
	uauth := umocks.NewAuthService(users, map[string][]umocks.SubjectSet{adminID: {uadminPolicy}})
	thingCache := mocks.NewCache()
	policiesCache := pmocks.NewCache()

	psvc := policies.NewService(uauth, pRepo, policiesCache, idProvider)

	csvc := clients.NewService(uauth, psvc, cRepo, gRepo, thingCache, idProvider)
	svc := groups.NewService(uauth, psvc, gRepo, idProvider)

	ts := newChannelsServer(csvc, svc, psvc)
	defer ts.Close()

	channel := sdk.Channel{
		Name:        "channelName",
		Description: description,
		Metadata:    validMetadata,
		Children:    []*sdk.Channel{},
		Status:      mfclients.EnabledStatus.String(),
	}

	conf := sdk.Config{
		ThingsURL: ts.URL,
	}
	mfsdk := sdk.NewSDK(conf)
	channel.ID = generateUUID(t)

	cases := []struct {
		desc      string
		token     string
		channelID string
		response  sdk.Channel
		err       errors.SDKError
	}{
		{
			desc:      "view channel",
			token:     adminToken,
			channelID: channel.ID,
			response:  channel,
			err:       nil,
		},
		{
			desc:      "view channel with invalid token",
			token:     "wrongtoken",
			channelID: channel.ID,
			response:  sdk.Channel{Children: []*sdk.Channel{}},
			err:       errors.NewSDKErrorWithStatus(errors.Wrap(errors.ErrAuthorization, errors.ErrAuthentication), http.StatusUnauthorized),
		},
		{
			desc:      "view channel for wrong id",
			token:     adminToken,
			channelID: gmocks.WrongID,
			response:  sdk.Channel{Children: []*sdk.Channel{}},
			err:       errors.NewSDKErrorWithStatus(errors.ErrNotFound, http.StatusNotFound),
		},
	}

	for _, tc := range cases {
		repoCall := pRepo.On("EvaluateGroupAccess", mock.Anything, mock.Anything).Return(policies.Policy{}, nil)
		repoCall1 := gRepo.On("RetrieveByID", mock.Anything, tc.channelID).Return(convertChannel(tc.response), tc.err)
		grp, err := mfsdk.Channel(tc.channelID, tc.token)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
		if len(tc.response.Children) == 0 {
			tc.response.Children = nil
		}
		if len(grp.Children) == 0 {
			grp.Children = nil
		}
		assert.Equal(t, tc.response, grp, fmt.Sprintf("%s: expected metadata %v got %v\n", tc.desc, tc.response, grp))
		if tc.err == nil {
			ok := repoCall.Parent.AssertCalled(t, "EvaluateGroupAccess", mock.Anything, mock.Anything)
			assert.True(t, ok, fmt.Sprintf("EvaluateGroupAccess was not called on %s", tc.desc))
			ok = repoCall1.Parent.AssertCalled(t, "RetrieveByID", mock.Anything, tc.channelID)
			assert.True(t, ok, fmt.Sprintf("RetrieveByID was not called on %s", tc.desc))
		}
		repoCall.Unset()
		repoCall1.Unset()
	}
}

func TestUpdateChannel(t *testing.T) {
	cRepo := new(mocks.Repository)
	gRepo := new(gmocks.Repository)
	pRepo := new(pmocks.Repository)
	uauth := umocks.NewAuthService(users, map[string][]umocks.SubjectSet{adminID: {uadminPolicy}})
	thingCache := mocks.NewCache()
	policiesCache := pmocks.NewCache()

	psvc := policies.NewService(uauth, pRepo, policiesCache, idProvider)

	csvc := clients.NewService(uauth, psvc, cRepo, gRepo, thingCache, idProvider)
	svc := groups.NewService(uauth, psvc, gRepo, idProvider)

	ts := newChannelsServer(csvc, svc, psvc)
	defer ts.Close()

	channel := sdk.Channel{
		ID:          generateUUID(t),
		Name:        "channelsName",
		Description: description,
		Metadata:    validMetadata,
	}

	conf := sdk.Config{
		ThingsURL: ts.URL,
	}
	mfsdk := sdk.NewSDK(conf)

	channel.ID = generateUUID(t)

	cases := []struct {
		desc     string
		token    string
		channel  sdk.Channel
		response sdk.Channel
		err      errors.SDKError
	}{
		{
			desc: "update channel name",
			channel: sdk.Channel{
				ID:   channel.ID,
				Name: "NewName",
			},
			response: sdk.Channel{
				ID:   channel.ID,
				Name: "NewName",
			},
			token: adminToken,
			err:   nil,
		},
		{
			desc: "update channel description",
			channel: sdk.Channel{
				ID:          channel.ID,
				Description: "NewDescription",
			},
			response: sdk.Channel{
				ID:          channel.ID,
				Description: "NewDescription",
			},
			token: adminToken,
			err:   nil,
		},
		{
			desc: "update channel metadata",
			channel: sdk.Channel{
				ID: channel.ID,
				Metadata: sdk.Metadata{
					"field": "value2",
				},
			},
			response: sdk.Channel{
				ID: channel.ID,
				Metadata: sdk.Metadata{
					"field": "value2",
				},
			},
			token: adminToken,
			err:   nil,
		},
		{
			desc: "update channel name with invalid channel id",
			channel: sdk.Channel{
				ID:   gmocks.WrongID,
				Name: "NewName",
			},
			response: sdk.Channel{},
			token:    adminToken,
			err:      errors.NewSDKErrorWithStatus(errors.ErrNotFound, http.StatusNotFound),
		},
		{
			desc: "update channel description with invalid channel id",
			channel: sdk.Channel{
				ID:          gmocks.WrongID,
				Description: "NewDescription",
			},
			response: sdk.Channel{},
			token:    adminToken,
			err:      errors.NewSDKErrorWithStatus(errors.ErrNotFound, http.StatusNotFound),
		},
		{
			desc: "update channel metadata with invalid channel id",
			channel: sdk.Channel{
				ID: gmocks.WrongID,
				Metadata: sdk.Metadata{
					"field": "value2",
				},
			},
			response: sdk.Channel{},
			token:    adminToken,
			err:      errors.NewSDKErrorWithStatus(errors.ErrNotFound, http.StatusNotFound),
		},
		{
			desc: "update channel name with invalid token",
			channel: sdk.Channel{
				ID:   channel.ID,
				Name: "NewName",
			},
			response: sdk.Channel{},
			token:    invalidToken,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(errors.ErrAuthorization, errors.ErrAuthentication), http.StatusUnauthorized),
		},
		{
			desc: "update channel description with invalid token",
			channel: sdk.Channel{
				ID:          channel.ID,
				Description: "NewDescription",
			},
			response: sdk.Channel{},
			token:    invalidToken,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(errors.ErrAuthorization, errors.ErrAuthentication), http.StatusUnauthorized),
		},
		{
			desc: "update channel metadata with invalid token",
			channel: sdk.Channel{
				ID: channel.ID,
				Metadata: sdk.Metadata{
					"field": "value2",
				},
			},
			response: sdk.Channel{},
			token:    invalidToken,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(errors.ErrAuthorization, errors.ErrAuthentication), http.StatusUnauthorized),
		},
		{
			desc: "update channel that can't be marshalled",
			channel: sdk.Channel{
				Name: "test",
				Metadata: map[string]interface{}{
					"test": make(chan int),
				},
			},
			response: sdk.Channel{},
			token:    token,
			err:      errors.NewSDKError(fmt.Errorf("json: unsupported type: chan int")),
		},
	}

	for _, tc := range cases {
		repoCall := pRepo.On("EvaluateGroupAccess", mock.Anything, mock.Anything).Return(policies.Policy{}, nil)
		repoCall1 := gRepo.On("Update", mock.Anything, mock.Anything).Return(convertChannel(tc.response), tc.err)
		_, err := mfsdk.UpdateChannel(tc.channel, tc.token)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
		if tc.err == nil {
			ok := repoCall.Parent.AssertCalled(t, "EvaluateGroupAccess", mock.Anything, mock.Anything)
			assert.True(t, ok, fmt.Sprintf("EvaluateGroupAccess was not called on %s", tc.desc))
			ok = repoCall1.Parent.AssertCalled(t, "Update", mock.Anything, mock.Anything)
			assert.True(t, ok, fmt.Sprintf("Update was not called on %s", tc.desc))
		}
		repoCall.Unset()
		repoCall1.Unset()
	}
}

func TestListChannelsByThing(t *testing.T) {
	cRepo := new(mocks.Repository)
	gRepo := new(gmocks.Repository)
	pRepo := new(pmocks.Repository)
	uauth := umocks.NewAuthService(users, map[string][]umocks.SubjectSet{adminID: {uadminPolicy}})
	thingCache := mocks.NewCache()
	policiesCache := pmocks.NewCache()

	psvc := policies.NewService(uauth, pRepo, policiesCache, idProvider)

	csvc := clients.NewService(uauth, psvc, cRepo, gRepo, thingCache, idProvider)
	svc := groups.NewService(uauth, psvc, gRepo, idProvider)

	ts := newChannelsServer(csvc, svc, psvc)
	defer ts.Close()

	conf := sdk.Config{
		ThingsURL: ts.URL,
	}
	mfsdk := sdk.NewSDK(conf)

	nChannels := uint64(100)
	aChannels := []sdk.Channel{}

	for i := uint64(1); i < nChannels; i++ {
		channel := sdk.Channel{
			Name:     fmt.Sprintf("membership_%d@example.com", i),
			Metadata: sdk.Metadata{"role": "channel"},
			Status:   mfclients.EnabledStatus.String(),
		}
		aChannels = append(aChannels, channel)
	}

	cases := []struct {
		desc     string
		token    string
		clientID string
		page     sdk.PageMetadata
		response []sdk.Channel
		err      errors.SDKError
	}{
		{
			desc:     "list channel with authorized token",
			token:    adminToken,
			clientID: testsutil.GenerateUUID(t, idProvider),
			page:     sdk.PageMetadata{},
			response: aChannels,
			err:      nil,
		},
		{
			desc:     "list channel with offset and limit",
			token:    adminToken,
			clientID: testsutil.GenerateUUID(t, idProvider),
			page: sdk.PageMetadata{
				Offset: 6,
				Total:  nChannels,
				Limit:  nChannels,
				Status: mfclients.AllStatus.String(),
			},
			response: aChannels[6 : nChannels-1],
			err:      nil,
		},
		{
			desc:     "list channel with given name",
			token:    adminToken,
			clientID: testsutil.GenerateUUID(t, idProvider),
			page: sdk.PageMetadata{
				Name:   gName,
				Offset: 6,
				Total:  nChannels,
				Limit:  nChannels,
				Status: mfclients.AllStatus.String(),
			},
			response: aChannels[6 : nChannels-1],
			err:      nil,
		},
		{
			desc:     "list channel with given level",
			token:    adminToken,
			clientID: testsutil.GenerateUUID(t, idProvider),
			page: sdk.PageMetadata{
				Level:  1,
				Offset: 6,
				Total:  nChannels,
				Limit:  nChannels,
				Status: mfclients.AllStatus.String(),
			},
			response: aChannels[6 : nChannels-1],
			err:      nil,
		},
		{
			desc:     "list channel with metadata",
			token:    adminToken,
			clientID: testsutil.GenerateUUID(t, idProvider),
			page: sdk.PageMetadata{
				Metadata: validMetadata,
				Offset:   6,
				Total:    nChannels,
				Limit:    nChannels,
				Status:   mfclients.AllStatus.String(),
			},
			response: aChannels[6 : nChannels-1],
			err:      nil,
		},
		{
			desc:     "list channel with an invalid token",
			token:    invalidToken,
			clientID: testsutil.GenerateUUID(t, idProvider),
			page:     sdk.PageMetadata{},
			response: []sdk.Channel(nil),
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(errors.ErrAuthorization, errors.ErrAuthentication), http.StatusUnauthorized),
		},
		{
			desc:     "list channel with an invalid id",
			token:    adminToken,
			clientID: gmocks.WrongID,
			page:     sdk.PageMetadata{},
			response: []sdk.Channel(nil),
			err:      errors.NewSDKErrorWithStatus(errors.ErrNotFound, http.StatusNotFound),
		},
	}

	for _, tc := range cases {
		repoCall := pRepo.On("EvaluateGroupAccess", mock.Anything, mock.Anything).Return(policies.Policy{}, nil)
		repoCall1 := gRepo.On("Memberships", mock.Anything, tc.clientID, mock.Anything).Return(convertChannelsMembershipPage(sdk.ChannelsPage{Channels: tc.response}), tc.err)
		page, err := mfsdk.ChannelsByThing(tc.clientID, tc.page, tc.token)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, page.Channels, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, page.Channels))
		if tc.err == nil {
			ok := repoCall.Parent.AssertCalled(t, "EvaluateGroupAccess", mock.Anything, mock.Anything)
			assert.True(t, ok, fmt.Sprintf("EvaluateGroupAccess was not called on %s", tc.desc))
			ok = repoCall1.Parent.AssertCalled(t, "Memberships", mock.Anything, tc.clientID, mock.Anything)
			assert.True(t, ok, fmt.Sprintf("Memberships was not called on %s", tc.desc))
		}
		repoCall.Unset()
		repoCall1.Unset()
	}
}

func TestEnableChannel(t *testing.T) {
	cRepo := new(mocks.Repository)
	gRepo := new(gmocks.Repository)
	pRepo := new(pmocks.Repository)
	uauth := umocks.NewAuthService(users, map[string][]umocks.SubjectSet{adminID: {uadminPolicy}})
	thingCache := mocks.NewCache()
	policiesCache := pmocks.NewCache()

	psvc := policies.NewService(uauth, pRepo, policiesCache, idProvider)

	csvc := clients.NewService(uauth, psvc, cRepo, gRepo, thingCache, idProvider)
	svc := groups.NewService(uauth, psvc, gRepo, idProvider)

	ts := newChannelsServer(csvc, svc, psvc)
	defer ts.Close()

	conf := sdk.Config{
		ThingsURL: ts.URL,
	}
	mfsdk := sdk.NewSDK(conf)

	creationTime := time.Now().UTC()
	channel := sdk.Channel{
		ID:        generateUUID(t),
		Name:      gName,
		OwnerID:   generateUUID(t),
		CreatedAt: creationTime,
		UpdatedAt: creationTime,
		Status:    mfclients.Disabled,
	}

	repoCall := pRepo.On("EvaluateGroupAccess", mock.Anything, mock.Anything).Return(policies.Policy{}, nil)
	repoCall1 := gRepo.On("RetrieveByID", mock.Anything, mock.Anything).Return(nil)
	repoCall2 := gRepo.On("ChangeStatus", mock.Anything, mock.Anything).Return(nil)
	_, err := mfsdk.EnableChannel("wrongID", adminToken)
	assert.Equal(t, err, errors.NewSDKErrorWithStatus(errors.Wrap(mfgroups.ErrEnableGroup, errors.ErrNotFound), http.StatusNotFound), fmt.Sprintf("Enable channel with wrong id: expected %v got %v", errors.ErrNotFound, err))
	ok := repoCall.Parent.AssertCalled(t, "EvaluateGroupAccess", mock.Anything, mock.Anything)
	assert.True(t, ok, "EvaluateGroupAccess was not called on enabling channel")
	ok = repoCall1.Parent.AssertCalled(t, "RetrieveByID", mock.Anything, "wrongID")
	assert.True(t, ok, "RetrieveByID was not called on enabling channel")
	repoCall.Unset()
	repoCall1.Unset()
	repoCall2.Unset()

	ch := mfgroups.Group{
		ID:        channel.ID,
		Name:      channel.Name,
		Owner:     channel.OwnerID,
		CreatedAt: creationTime,
		UpdatedAt: creationTime,
		Status:    mfclients.DisabledStatus,
	}

	repoCall = pRepo.On("EvaluateGroupAccess", mock.Anything, mock.Anything).Return(policies.Policy{}, nil)
	repoCall1 = gRepo.On("RetrieveByID", mock.Anything, mock.Anything).Return(ch, nil)
	repoCall2 = gRepo.On("ChangeStatus", mock.Anything, mock.Anything).Return(ch, nil)
	res, err := mfsdk.EnableChannel(channel.ID, adminToken)
	assert.Nil(t, err, fmt.Sprintf("Enable channel with correct id: expected %v got %v", nil, err))
	assert.Equal(t, channel, res, fmt.Sprintf("Enable channel with correct id: expected %v got %v", channel, res))
	ok = repoCall.Parent.AssertCalled(t, "EvaluateGroupAccess", mock.Anything, mock.Anything)
	assert.True(t, ok, "EvaluateGroupAccess was not called on enabling channel")
	ok = repoCall1.Parent.AssertCalled(t, "RetrieveByID", mock.Anything, channel.ID)
	assert.True(t, ok, "RetrieveByID was not called on enabling channel")
	ok = repoCall2.Parent.AssertCalled(t, "ChangeStatus", mock.Anything, mock.Anything)
	assert.True(t, ok, "ChangeStatus was not called on enabling channel")
	repoCall.Unset()
	repoCall1.Unset()
	repoCall2.Unset()
}

func TestDisableChannel(t *testing.T) {
	cRepo := new(mocks.Repository)
	gRepo := new(gmocks.Repository)
	pRepo := new(pmocks.Repository)
	uauth := umocks.NewAuthService(users, map[string][]umocks.SubjectSet{adminID: {uadminPolicy}})
	thingCache := mocks.NewCache()
	policiesCache := pmocks.NewCache()

	psvc := policies.NewService(uauth, pRepo, policiesCache, idProvider)

	csvc := clients.NewService(uauth, psvc, cRepo, gRepo, thingCache, idProvider)
	svc := groups.NewService(uauth, psvc, gRepo, idProvider)

	ts := newChannelsServer(csvc, svc, psvc)
	defer ts.Close()

	conf := sdk.Config{
		ThingsURL: ts.URL,
	}
	mfsdk := sdk.NewSDK(conf)

	creationTime := time.Now().UTC()
	channel := sdk.Channel{
		ID:        generateUUID(t),
		Name:      gName,
		OwnerID:   generateUUID(t),
		CreatedAt: creationTime,
		UpdatedAt: creationTime,
		Status:    mfclients.Enabled,
	}

	repoCall := pRepo.On("EvaluateGroupAccess", mock.Anything, mock.Anything).Return(policies.Policy{}, nil)
	repoCall1 := gRepo.On("ChangeStatus", mock.Anything, mock.Anything).Return(sdk.ErrFailedRemoval)
	repoCall2 := gRepo.On("RetrieveByID", mock.Anything, mock.Anything).Return(nil)
	_, err := mfsdk.DisableChannel("wrongID", adminToken)
	assert.Equal(t, err, errors.NewSDKErrorWithStatus(errors.Wrap(mfgroups.ErrDisableGroup, errors.ErrNotFound), http.StatusNotFound), fmt.Sprintf("Disable channel with wrong id: expected %v got %v", errors.ErrNotFound, err))
	ok := repoCall.Parent.AssertCalled(t, "EvaluateGroupAccess", mock.Anything, mock.Anything)
	assert.True(t, ok, "EvaluateGroupAccess was not called on disabling group with wrong id")
	ok = repoCall1.Parent.AssertCalled(t, "RetrieveByID", mock.Anything, "wrongID")
	assert.True(t, ok, "Memberships was not called on disabling channel with wrong id")
	repoCall.Unset()
	repoCall1.Unset()
	repoCall2.Unset()

	ch := mfgroups.Group{
		ID:        channel.ID,
		Name:      channel.Name,
		Owner:     channel.OwnerID,
		CreatedAt: creationTime,
		UpdatedAt: creationTime,
		Status:    mfclients.EnabledStatus,
	}

	repoCall = pRepo.On("EvaluateGroupAccess", mock.Anything, mock.Anything).Return(policies.Policy{}, nil)
	repoCall1 = gRepo.On("ChangeStatus", mock.Anything, mock.Anything).Return(ch, nil)
	repoCall2 = gRepo.On("RetrieveByID", mock.Anything, mock.Anything).Return(ch, nil)
	res, err := mfsdk.DisableChannel(channel.ID, adminToken)
	assert.Nil(t, err, fmt.Sprintf("Disable channel with correct id: expected %v got %v", nil, err))
	assert.Equal(t, channel, res, fmt.Sprintf("Disable channel with correct id: expected %v got %v", channel, res))
	ok = repoCall.Parent.AssertCalled(t, "EvaluateGroupAccess", mock.Anything, mock.Anything)
	assert.True(t, ok, "EvaluateGroupAccess was not called on disabling channel with correct id")
	ok = repoCall1.Parent.AssertCalled(t, "RetrieveByID", mock.Anything, channel.ID)
	assert.True(t, ok, "RetrieveByID was not called on disabling channel with correct id")
	ok = repoCall2.Parent.AssertCalled(t, "ChangeStatus", mock.Anything, mock.Anything)
	assert.True(t, ok, "ChangeStatus was not called on disabling channel with correct id")
	repoCall.Unset()
	repoCall1.Unset()
	repoCall2.Unset()
}
