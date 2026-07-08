package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

func main() {
	url := "http://localhost:8085/api/v1/messaging/send"
	apiKey := "hh_live_bd0867efabaf4b6cb1d46b897926b328"
	
	payload := map[string]string{
		"phone_number": "+967777217167",
		"content":      "الحمد لله! تجربة إرسال رسالة ناجحة بعد إصلاح قاعدة البيانات وتحديث كود المزامنة للتقارير المالية.",
	}
	
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		fmt.Println("Error marshaling payload:", err)
		return
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		fmt.Println("Error creating request:", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", apiKey)
	req.Header.Set("X-Idempotency-Key", fmt.Sprintf("sim_%d", time.Now().UnixNano()))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request:", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response:", err)
		return
	}

	fmt.Println("Response Status:", resp.Status)
	fmt.Println("Response Body:", string(body))
}
