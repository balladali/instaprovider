package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/siongui/instago"
	"net/http"
	"runtime/debug"
	"strconv"
	"strings"
)

type InstaPost struct {
	Description string   `json:"description"`
	PhotoURLs   []string `json:"photo_urls"`
	VideoURLs   []string `json:"video_urls"`
	Likes       int64    `json:"likes"`
	ID          string   `json:"id"`
	PostUrl     string   `json:"post_url"`
}

type InstaUser struct {
	UserName string       `json:"user_name"`
	Posts    []InstaPost  `json:"posts"`
	Stories  []InstaStory `json:"stories"`
}

type InstaStory struct {
	StoryURL   string `json:"story_url"`
	OriginalID string `json:"original_id"`
	ID         string `json:"id"`
	MediaURL   string `json:"media_url"`
}

type InstagramHandler struct {
	PrivateAPIManager instago.IGApiManager
	PublicAPIManager  instago.IGApiManager
}

func newInstagramHandler(dcUserId string, sessionID string, csrfToken string) InstagramHandler {
	privateAPIManager := *instago.NewInstagramApiManager(dcUserId, sessionID, csrfToken)
	publicAPIManager := *instago.NewInstagramApiManager("", "", "")
	return InstagramHandler{
		PrivateAPIManager: privateAPIManager,
		PublicAPIManager:  publicAPIManager,
	}
}

func (handler InstagramHandler) handlePostsRequest() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if r := recover(); r != nil {
				w.WriteHeader(500)
				w.Write([]byte(fmt.Sprintf("{\"Error\": \"%+v\"}", r)))
				fmt.Println("recovered from ", r)
				debug.PrintStack()
			}
		}()
		vars := mux.Vars(r)
		name := vars["username"]
		id := getInt(vars["last"], 0)

		resp := handler.getPosts(name, id)
		res, _ := json.Marshal(resp)
		w.WriteHeader(200)
		w.Write(res)
	}
}

func (handler InstagramHandler) getPosts(name string, id int64) *InstaUser {
	userInfo, _ := handler.PublicAPIManager.GetUserInfo(name)
	var medias []instago.IGMedia
	if userInfo.IsPrivate {
		medias, _ = handler.PrivateAPIManager.GetAllPostMedia(name)
	} else {
		medias, _ = handler.PublicAPIManager.GetAllPostMedia(name)
	}
	resp := &InstaUser{UserName: name, Posts: []InstaPost{}}
	for _, media := range medias {
		mediaId := getInt(media.Id, 0)
		if mediaId <= id {
			break
		}
		postInfo := InstaPost{
			PostUrl:  media.GetPostUrl(),
			Likes:    media.EdgeMediaPreviewLike.Count,
			ID:       media.Id,
		}
		if len(media.EdgeMediaToCaption.Edges) > 0 {
			postInfo.Description = media.EdgeMediaToCaption.Edges[0].Node.Text
		}
		if media.IsVideo {
			//all posts public response does not contain video url
			info, _ := handler.PrivateAPIManager.GetPostInfo(media.GetPostCode())
			postInfo.VideoURLs = info.GetMediaUrls()
		} else {
			info, _ := handler.PublicAPIManager.GetPostInfo(media.GetPostCode())
			postInfo.PhotoURLs = info.GetMediaUrls()
		}
		resp.Posts = append(resp.Posts, postInfo)
	}
	return resp
}

func (handler InstagramHandler) handleStoriesRequest() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if r := recover(); r != nil {
				w.WriteHeader(500)
				w.Write([]byte(fmt.Sprintf("{\"Error\": \"%+v\"}", r)))
				fmt.Println("recovered from ", r)
				debug.PrintStack()
			}
		}()
		vars := mux.Vars(r)
		name := vars["username"]
		last := vars["last"]
		lastId := getInt(last, 0)

		resp := handler.getStories(name, lastId)
		res, _ := json.Marshal(resp)
		w.WriteHeader(200)
		w.Write(res)
	}
}

func (handler InstagramHandler) getStories(name string, lastId int64) *InstaUser {
	userId, _ := instago.GetUserId(name)
	stories, _ := handler.PrivateAPIManager.GetUserStory(userId)
	resp := &InstaUser{UserName: name, Stories: []InstaStory{}}
	for _, story := range stories.GetItems() {
		storyId := getStoryIdWithoutUserId(story.Id)
		if storyId <= lastId {
			continue
		}
		media, _ := story.GetMediaUrls()
		storyInfo := InstaStory{
			StoryURL:   story.GetPostUrl(),
			ID:         strconv.FormatInt(storyId, 10),
			OriginalID: story.Id,
			MediaURL:   media[0],
		}
		resp.Stories = append(resp.Stories, storyInfo)
	}
	return resp
}

func getStoryIdWithoutUserId(storyId string) int64 {
	storyIdString := storyId[:strings.IndexByte(storyId, '_')]
	return getInt(storyIdString, 0)
}

func getInt(strValue string, defaultValue int64) int64 {
	intValue, err := strconv.ParseInt(strValue, 10, 64)
	if err != nil {
		fmt.Printf("Incorrect int value, default value %+v will be used ", defaultValue)
		return defaultValue
	}
	return intValue
}
