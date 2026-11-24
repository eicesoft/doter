package translate

import "testing"

func TestParseGoogle(t *testing.T) {
	body := []byte(`[[["你好世界","hello world",null,null,3],["，你好吗","，how are you",null,null,3]],null,"en",null]`)
	res, err := parseGoogle(body, "zh-CN")
	if err != nil {
		t.Fatalf("parseGoogle: %v", err)
	}
	if res.Text != "你好世界，你好吗" {
		t.Fatalf("text = %q", res.Text)
	}
	if res.Source != "en" || res.Target != "zh-CN" || res.Provider != "Google" {
		t.Fatalf("meta = %+v", res)
	}

	if _, err := parseGoogle([]byte(`[null]`), "en"); err == nil {
		t.Fatal("expected an error for a malformed payload")
	}
}

func TestParseMyMemory(t *testing.T) {
	body := []byte(`{"responseData":{"translatedText":"Hello","detectedLanguage":"zh-CN"},"responseStatus":200}`)
	res, err := parseMyMemory(body, "en")
	if err != nil {
		t.Fatalf("parseMyMemory: %v", err)
	}
	if res.Text != "Hello" || res.Source != "zh-CN" || res.Provider != "MyMemory" {
		t.Fatalf("result = %+v", res)
	}

	body = []byte(`{"responseData":{"translatedText":""},"responseStatus":200}`)
	if _, err := parseMyMemory(body, "en"); err == nil {
		t.Fatal("expected an error for an empty translation")
	}
}
