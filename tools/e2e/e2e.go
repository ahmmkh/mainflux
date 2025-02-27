// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"time"

	"github.com/gookit/color"
	namegen "github.com/goombaio/namegenerator"
	"github.com/gorilla/websocket"
	sdk "github.com/mainflux/mainflux/pkg/sdk/go"
	"golang.org/x/sync/errgroup"
)

const (
	defPass      = "12345678"
	defReaderURL = "http://localhost:8905"
	defWSPort    = "8186"
	numAdapters  = 4
)

var (
	seed           = time.Now().UTC().UnixNano()
	namesgenerator = namegen.NewNameGenerator(seed)
	msgFormat      = `[{"bn":"demo", "bu":"V", "t": %d, "bver":5, "n":"voltage", "u":"V", "v":%d}]`
)

// Config - test configuration.
type Config struct {
	Host     string
	Num      uint64
	NumOfMsg uint64
	SSL      bool
	CA       string
	CAKey    string
	Prefix   string
}

func init() {
	rand.Seed(seed)
}

// Test - function that does actual end to end testing.
// The operations are:
// - Create a user
// - Create other users
// - Do Read, Update and Change of Status operations on users.

// - Create groups using hierarchy
// - Do Read, Update and Change of Status operations on groups.

// - Create things
// - Do Read, Update and Change of Status operations on things.

// - Create channels
// - Do Read, Update and Change of Status operations on channels.

// - Connect thing to channel
// - Publish message from HTTP, MQTT, WS and CoAP Adapters.
func Test(conf Config) {
	sdkConf := sdk.Config{
		ThingsURL:       fmt.Sprintf("http://%s", conf.Host),
		UsersURL:        fmt.Sprintf("http://%s", conf.Host),
		ReaderURL:       defReaderURL,
		HTTPAdapterURL:  fmt.Sprintf("http://%s/http", conf.Host),
		BootstrapURL:    fmt.Sprintf("http://%s", conf.Host),
		CertsURL:        fmt.Sprintf("http://%s", conf.Host),
		MsgContentType:  sdk.CTJSONSenML,
		TLSVerification: false,
	}

	s := sdk.NewSDK(sdkConf)

	magenta := color.FgLightMagenta.Render

	token, err := createUser(s, conf)
	if err != nil {
		errExit(fmt.Errorf("unable to create user: %w", err))
	}
	color.Success.Printf("created user with token %s\n", magenta(token))

	users, err := createUsers(s, conf, token)
	if err != nil {
		errExit(fmt.Errorf("unable to create users: %w", err))
	}
	color.Success.Printf("created users of ids:\n%s\n", magenta(getIDS(users)))

	groups, err := createGroups(s, conf, token)
	if err != nil {
		errExit(fmt.Errorf("unable to create groups: %w", err))
	}
	color.Success.Printf("created groups of ids:\n%s\n", magenta(getIDS(groups)))

	things, err := createThings(s, conf, token)
	if err != nil {
		errExit(fmt.Errorf("unable to create things: %w", err))
	}
	color.Success.Printf("created things of ids:\n%s\n", magenta(getIDS(things)))

	channels, err := createChannels(s, conf, token)
	if err != nil {
		errExit(fmt.Errorf("unable to create channels: %w", err))
	}
	color.Success.Printf("created channels of ids:\n%s\n", magenta(getIDS(channels)))

	// List users, groups, things and channels
	if err := read(s, conf, token, users, groups, things, channels); err != nil {
		errExit(fmt.Errorf("unable to read users, groups, things and channels: %w", err))
	}
	color.Success.Println("viewed users, groups, things and channels")

	// Update users, groups, things and channels
	if err := update(s, token, users, groups, things, channels); err != nil {
		errExit(fmt.Errorf("unable to update users, groups, things and channels: %w", err))
	}
	color.Success.Println("updated users, groups, things and channels")

	// Send messages to channels
	if err := messaging(s, conf, token, things, channels); err != nil {
		errExit(fmt.Errorf("unable to send messages to channels: %w", err))
	}
	color.Success.Println("sent messages to channels")
}

func errExit(err error) {
	color.Error.Println(err.Error())
	os.Exit(1)
}

func createUser(s sdk.SDK, conf Config) (string, error) {
	user := sdk.User{
		Name: fmt.Sprintf("%s-%s", conf.Prefix, namesgenerator.Generate()),
		Credentials: sdk.Credentials{
			Identity: fmt.Sprintf("%s-%s@email.com", conf.Prefix, namesgenerator.Generate()),
			Secret:   defPass,
		},
		Status: sdk.EnabledStatus,
	}

	pass := user.Credentials.Secret

	user, err := s.CreateUser(user, "")
	if err != nil {
		return "", fmt.Errorf("unable to create user: %w", err)
	}

	user.Credentials.Secret = pass
	token, err := s.CreateToken(user)
	if err != nil {
		return "", fmt.Errorf("unable to login user: %w", err)
	}

	return token.AccessToken, nil
}

func createUsers(s sdk.SDK, conf Config, token string) ([]sdk.User, error) {
	var err error
	users := []sdk.User{}

	for i := uint64(0); i < conf.Num; i++ {
		user := sdk.User{
			Name: fmt.Sprintf("%s-%s", conf.Prefix, namesgenerator.Generate()),
			Credentials: sdk.Credentials{
				Identity: fmt.Sprintf("%s-%s@email.com", conf.Prefix, namesgenerator.Generate()),
				Secret:   defPass,
			},
			Status: sdk.EnabledStatus,
		}

		user, err = s.CreateUser(user, token)
		if err != nil {
			return []sdk.User{}, fmt.Errorf("Failed to create the users: %w", err)
		}
		users = append(users, user)
	}

	return users, nil
}

func createGroups(s sdk.SDK, conf Config, token string) ([]sdk.Group, error) {
	var err error
	groups := []sdk.Group{}

	parentID := ""
	for i := uint64(0); i < conf.Num; i++ {
		group := sdk.Group{
			Name:     fmt.Sprintf("%s-%s", conf.Prefix, namesgenerator.Generate()),
			ParentID: parentID,
			Status:   sdk.EnabledStatus,
		}

		group, err = s.CreateGroup(group, token)
		if err != nil {
			return []sdk.Group{}, fmt.Errorf("Failed to create the group: %w", err)
		}
		groups = append(groups, group)
		parentID = group.ID
	}

	return groups, nil
}

func createThings(s sdk.SDK, conf Config, token string) ([]sdk.Thing, error) {
	var err error
	things := make([]sdk.Thing, conf.Num)

	for i := uint64(0); i < conf.Num; i++ {
		things[i] = sdk.Thing{
			Name:   fmt.Sprintf("%s-%s", conf.Prefix, namesgenerator.Generate()),
			Status: sdk.EnabledStatus,
		}
	}
	things, err = s.CreateThings(things, token)
	if err != nil {
		return []sdk.Thing{}, fmt.Errorf("Failed to create the things: %w", err)
	}

	return things, nil
}

func createChannels(s sdk.SDK, conf Config, token string) ([]sdk.Channel, error) {
	var err error
	channels := make([]sdk.Channel, conf.Num)

	for i := uint64(0); i < conf.Num; i++ {
		channels[i] = sdk.Channel{
			Name:   fmt.Sprintf("%s-%s", conf.Prefix, namesgenerator.Generate()),
			Status: sdk.EnabledStatus,
		}
	}

	channels, err = s.CreateChannels(channels, token)
	if err != nil {
		return []sdk.Channel{}, fmt.Errorf("Failed to create the channels: %w", err)
	}

	return channels, nil
}

func read(s sdk.SDK, conf Config, token string, users []sdk.User, groups []sdk.Group, things []sdk.Thing, channels []sdk.Channel) error {
	for _, user := range users {
		if _, err := s.User(user.ID, token); err != nil {
			return err
		}
	}
	up, err := s.Users(sdk.PageMetadata{}, token)
	if err != nil {
		return err
	}
	if up.Total != conf.Num {
		return fmt.Errorf("returned users %d not equal to create users %d", up.Total, conf.Num)
	}
	for _, group := range groups {
		if _, err := s.Group(group.ID, token); err != nil {
			return err
		}
	}
	gp, err := s.Groups(sdk.PageMetadata{}, token)
	if err != nil {
		return err
	}
	if gp.Total != conf.Num {
		return fmt.Errorf("returned groups %d not equal to create groups %d", gp.Total, conf.Num)
	}
	for _, thing := range things {
		if _, err := s.Thing(thing.ID, token); err != nil {
			return err
		}
	}
	tp, err := s.Things(sdk.PageMetadata{}, token)
	if err != nil {
		return err
	}
	if tp.Total != conf.Num {
		return fmt.Errorf("returned things %d not equal to create things %d", tp.Total, conf.Num)
	}
	for _, channel := range channels {
		if _, err := s.Channel(channel.ID, token); err != nil {
			return err
		}
	}
	cp, err := s.Channels(sdk.PageMetadata{}, token)
	if err != nil {
		return err
	}
	if cp.Total != conf.Num {
		return fmt.Errorf("returned channels %d not equal to create channels %d", cp.Total, conf.Num)
	}

	return nil
}

func update(s sdk.SDK, token string, users []sdk.User, groups []sdk.Group, things []sdk.Thing, channels []sdk.Channel) error {
	for _, user := range users {
		user.Name = namesgenerator.Generate()
		user.Metadata = sdk.Metadata{"Update": namesgenerator.Generate()}
		rUser, err := s.UpdateUser(user, token)
		if err != nil {
			return fmt.Errorf("failed to update user %w", err)
		}
		if rUser.Name != user.Name {
			return fmt.Errorf("failed to update user name before %s after %s", user.Name, rUser.Name)
		}
		if rUser.Metadata["Update"] != user.Metadata["Update"] {
			return fmt.Errorf("failed to update user metadata before %s after %s", user.Metadata["Update"], rUser.Metadata["Update"])
		}
		user = rUser
		user.Credentials.Identity = namesgenerator.Generate()
		rUser, err = s.UpdateUserIdentity(user, token)
		if err != nil {
			return fmt.Errorf("failed to update user identity %w", err)
		}
		if rUser.Credentials.Identity != user.Credentials.Identity {
			return fmt.Errorf("failed to update user identity before %s after %s", user.Credentials.Identity, rUser.Credentials.Identity)
		}
		user = rUser
		user.Tags = []string{namesgenerator.Generate()}
		rUser, err = s.UpdateUserTags(user, token)
		if err != nil {
			return fmt.Errorf("failed to update user tags %w", err)
		}
		if rUser.Tags[0] != user.Tags[0] {
			return fmt.Errorf("failed to update user tags before %s after %s", user.Tags[0], rUser.Tags[0])
		}
		user = rUser
		rUser, err = s.DisableUser(user.ID, token)
		if err != nil {
			return fmt.Errorf("failed to disable user %w", err)
		}
		if rUser.Status != sdk.DisabledStatus {
			return fmt.Errorf("failed to disable user before %s after %s", user.Status, rUser.Status)
		}
		user = rUser
		rUser, err = s.EnableUser(user.ID, token)
		if err != nil {
			return fmt.Errorf("failed to enable user %w", err)
		}
		if rUser.Status != sdk.EnabledStatus {
			return fmt.Errorf("failed to enable user before %s after %s", user.Status, rUser.Status)
		}
	}
	for _, group := range groups {
		group.Name = namesgenerator.Generate()
		group.Metadata = sdk.Metadata{"Update": namesgenerator.Generate()}
		rGroup, err := s.UpdateGroup(group, token)
		if err != nil {
			return fmt.Errorf("failed to update group %w", err)
		}
		if rGroup.Name != group.Name {
			return fmt.Errorf("failed to update group name before %s after %s", group.Name, rGroup.Name)
		}
		if rGroup.Metadata["Update"] != group.Metadata["Update"] {
			return fmt.Errorf("failed to update group metadata before %s after %s", group.Metadata["Update"], rGroup.Metadata["Update"])
		}
		group = rGroup
		rGroup, err = s.DisableGroup(group.ID, token)
		if err != nil {
			return fmt.Errorf("failed to disable group %w", err)
		}
		if rGroup.Status != sdk.DisabledStatus {
			return fmt.Errorf("failed to disable group before %s after %s", group.Status, rGroup.Status)
		}
		group = rGroup
		rGroup, err = s.EnableGroup(group.ID, token)
		if err != nil {
			return fmt.Errorf("failed to enable group %w", err)
		}
		if rGroup.Status != sdk.EnabledStatus {
			return fmt.Errorf("failed to enable group before %s after %s", group.Status, rGroup.Status)
		}
	}
	for _, thing := range things {
		thing.Name = namesgenerator.Generate()
		thing.Metadata = sdk.Metadata{"Update": namesgenerator.Generate()}
		rThing, err := s.UpdateThing(thing, token)
		if err != nil {
			return fmt.Errorf("failed to update thing %w", err)
		}
		if rThing.Name != thing.Name {
			return fmt.Errorf("failed to update thing name before %s after %s", thing.Name, rThing.Name)
		}
		if rThing.Metadata["Update"] != thing.Metadata["Update"] {
			return fmt.Errorf("failed to update thing metadata before %s after %s", thing.Metadata["Update"], rThing.Metadata["Update"])
		}
		thing = rThing
		rThing, err = s.UpdateThingSecret(thing.ID, thing.Credentials.Secret, token)
		if err != nil {
			return fmt.Errorf("failed to update thing secret %w", err)
		}
		if rThing.Credentials.Secret != thing.Credentials.Secret {
			return fmt.Errorf("failed to update thing secret before %s after %s", thing.Credentials.Secret, rThing.Credentials.Secret)
		}
		thing = rThing
		thing.Tags = []string{namesgenerator.Generate()}
		rThing, err = s.UpdateThingTags(thing, token)
		if err != nil {
			return fmt.Errorf("failed to update thing tags %w", err)
		}
		if rThing.Tags[0] != thing.Tags[0] {
			return fmt.Errorf("failed to update thing tags before %s after %s", thing.Tags[0], rThing.Tags[0])
		}
		thing = rThing
		rThing, err = s.DisableThing(thing.ID, token)
		if err != nil {
			return fmt.Errorf("failed to disable thing %w", err)
		}
		if rThing.Status != sdk.DisabledStatus {
			return fmt.Errorf("failed to disable thing before %s after %s", thing.Status, rThing.Status)
		}
		thing = rThing
		rThing, err = s.EnableThing(thing.ID, token)
		if err != nil {
			return fmt.Errorf("failed to enable thing %w", err)
		}
		if rThing.Status != sdk.EnabledStatus {
			return fmt.Errorf("failed to enable thing before %s after %s", thing.Status, rThing.Status)
		}
	}
	for _, channel := range channels {
		channel.Name = namesgenerator.Generate()
		channel.Metadata = sdk.Metadata{"Update": namesgenerator.Generate()}
		rChannel, err := s.UpdateChannel(channel, token)
		if err != nil {
			return fmt.Errorf("failed to update channel %w", err)
		}
		if rChannel.Name != channel.Name {
			return fmt.Errorf("failed to update channel name before %s after %s", channel.Name, rChannel.Name)
		}
		if rChannel.Metadata["Update"] != channel.Metadata["Update"] {
			return fmt.Errorf("failed to update channel metadata before %s after %s", channel.Metadata["Update"], rChannel.Metadata["Update"])
		}
		channel = rChannel
		rChannel, err = s.DisableChannel(channel.ID, token)
		if err != nil {
			return fmt.Errorf("failed to disable channel %w", err)
		}
		if rChannel.Status != sdk.DisabledStatus {
			return fmt.Errorf("failed to disable channel before %s after %s", channel.Status, rChannel.Status)
		}
		channel = rChannel
		rChannel, err = s.EnableChannel(channel.ID, token)
		if err != nil {
			return fmt.Errorf("failed to enable channel %w", err)
		}
		if rChannel.Status != sdk.EnabledStatus {
			return fmt.Errorf("failed to enable channel before %s after %s", channel.Status, rChannel.Status)
		}
	}

	return nil
}

func messaging(s sdk.SDK, conf Config, token string, things []sdk.Thing, channels []sdk.Channel) error {
	for _, thing := range things {
		for _, channel := range channels {
			if err := s.ConnectThing(thing.ID, channel.ID, token); err != nil {
				return fmt.Errorf("failed to connect thing %s to channel %s", thing.ID, channel.ID)
			}
		}
	}
	g := new(errgroup.Group)

	bt := time.Now().Unix()
	for i := uint64(0); i < conf.NumOfMsg; i++ {
		for _, thing := range things {
			for _, channel := range channels {
				func(num int64, thing sdk.Thing, channel sdk.Channel) {
					g.Go(func() error {
						msg := fmt.Sprintf(msgFormat, num+1, rand.Int())
						return sendHTTPMessage(s, msg, thing, channel.ID)
					})
					g.Go(func() error {
						msg := fmt.Sprintf(msgFormat, num+2, rand.Int())
						return sendCoAPMessage(msg, thing, channel.ID)
					})
					g.Go(func() error {
						msg := fmt.Sprintf(msgFormat, num+3, rand.Int())
						return sendMQTTMessage(msg, thing, channel.ID)
					})
					g.Go(func() error {
						msg := fmt.Sprintf(msgFormat, num+4, rand.Int())
						return sendWSMessage(conf, msg, thing, channel.ID)
					})
				}(bt, thing, channel)
				bt += numAdapters
			}
		}
	}

	return g.Wait()
}

func sendHTTPMessage(s sdk.SDK, msg string, thing sdk.Thing, chanID string) error {
	if err := s.SendMessage(chanID, msg, thing.Credentials.Secret); err != nil {
		return fmt.Errorf("HTTP failed to send message from thing %s to channel %s: %w", thing.ID, chanID, err)
	}

	return nil
}

func sendCoAPMessage(msg string, thing sdk.Thing, chanID string) error {
	cmd := exec.Command("coap-cli", "post", fmt.Sprintf("channels/%s/messages", chanID), "-auth", thing.Credentials.Secret, "-d", msg)
	if _, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("CoAP failed to send message from thing %s to channel %s: %w", thing.ID, chanID, err)
	}

	return nil
}

func sendMQTTMessage(msg string, thing sdk.Thing, chanID string) error {
	cmd := exec.Command("mosquitto_pub", "--id-prefix", "mainflux", "-u", thing.ID, "-P", thing.Credentials.Secret, "-t", fmt.Sprintf("channels/%s/messages", chanID), "-h", "localhost", "-m", msg)
	if _, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("MQTT failed to send message from thing %s to channel %s: %w", thing.ID, chanID, err)
	}

	return nil
}

func sendWSMessage(conf Config, msg string, thing sdk.Thing, chanID string) error {
	socketURL := fmt.Sprintf("ws://%s:%s/channels/%s/messages", conf.Host, defWSPort, chanID)
	header := http.Header{"authorization": []string{thing.Credentials.Secret}}
	conn, _, err := websocket.DefaultDialer.Dial(socketURL, header)
	if err != nil {
		return fmt.Errorf("unable to connect to websocket: %w", err)
	}
	defer conn.Close()
	if err := conn.WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
		return fmt.Errorf("WS failed to send message from thing %s to channel %s: %w", thing.ID, chanID, err)
	}

	return nil
}

// getIDS returns a list of IDs of the given objects.
func getIDS(objects interface{}) string {
	v := reflect.ValueOf(objects)
	if v.Kind() != reflect.Slice {
		panic("objects argument must be a slice")
	}
	ids := make([]string, v.Len())
	for i := 0; i < v.Len(); i++ {
		id := v.Index(i).FieldByName("ID").String()
		ids[i] = id
	}
	idList := strings.Join(ids, "\n")

	return idList
}
