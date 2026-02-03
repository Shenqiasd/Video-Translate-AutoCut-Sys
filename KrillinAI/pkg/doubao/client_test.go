package doubao

import (
	"testing"
)

func TestNewDoubaoClient(t *testing.T) {
	appId := "test_app_id"
	token := "test_token"
	cluster := "test_cluster"

	client := NewDoubaoClient(appId, token, cluster)

	if client.AppId != appId {
		t.Errorf("Expected AppId %s, got %s", appId, client.AppId)
	}
	if client.AccessToken != token {
		t.Errorf("Expected AccessToken %s, got %s", token, client.AccessToken)
	}
	if client.Cluster != cluster {
		t.Errorf("Expected Cluster %s, got %s", cluster, client.Cluster)
	}
	if client.BaseURL != "https://openspeech.bytedance.com/api/v1/tts" {
		t.Errorf("Expected BaseURL %s, got %s", "https://openspeech.bytedance.com/api/v1/tts", client.BaseURL)
	}
}

func TestNewDoubaoClientDefaultCluster(t *testing.T) {
	client := NewDoubaoClient("id", "token", "")
	if client.Cluster != "volcano_tts" {
		t.Errorf("Expected default cluster volcano_tts, got %s", client.Cluster)
	}
}
