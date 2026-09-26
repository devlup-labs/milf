package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

type CopilotRequest struct {
	Prompt string `json:"prompt"`
	Code   string `json:"code,omitempty"`
}

type GeminiContentPart struct {
	Text string `json:"text"`
}

type GeminiContent struct {
	Role  string              `json:"role,omitempty"`
	Parts []GeminiContentPart `json:"parts"`
}

type GeminiSystemInstruction struct {
	Parts []GeminiContentPart `json:"parts"`
}

type GeminiGenerationConfig struct {
	ResponseMimeType string `json:"responseMimeType,omitempty"`
}

type GeminiRequest struct {
	Contents          []GeminiContent          `json:"contents"`
	SystemInstruction GeminiSystemInstruction `json:"systemInstruction,omitempty"`
	GenerationConfig  GeminiGenerationConfig  `json:"generationConfig,omitempty"`
}

type GeminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

type CopilotResponse struct {
	Insight string `json:"insight"`
	Code    string `json:"code"`
}

func (h *LambdaHandler) HandleCopilot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed", "")
		return
	}

	var req CopilotRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}

	// 1. Prioritize user-provided key from header (BYOK), fallback to system env key
	apiKey := r.Header.Get("X-Gemini-Api-Key")
	if apiKey == "" {
		apiKey = os.Getenv("GEMINI_API_KEY")
	}

	if apiKey == "" {
		writeJSON(w, http.StatusOK, CopilotResponse{
			Insight: "- **Missing Gemini API Key**: Please configure your own Gemini API Key in the **Settings** dashboard to enable the live AI Copilot.\n- Showing a mock/example response to help you understand the format.",
			Code: `#include <milf.h>

// Example MILF implementation
MILF_EXPORT int wasm_main(char* payload, int payload_len, char* out_buf, int out_max) {
    const char* response = "{\"status\": \"ok\", \"info\": \"Please connect your Gemini Key in Settings to activate Copilot\"}";
    milf_memcpy(out_buf, response, milf_strlen(response));
    return milf_strlen(response);
}`,
		})
		return
	}

	systemText := `You are the MILF AI Copilot. MILF compiles C code to WebAssembly using clang with -nostdlib (no standard library) for execution on Android/Flutter nodes via WAMR.

CAUTION - READ BEFORE GENERATING CODE:
Standard AI generated C code WILL FAIL TO COMPILE here. 
- You do NOT have malloc, free, printf, snprintf, string.h, or stdio.h.
- If you need string formatting or memory allocation, you MUST implement basic versions (e.g., inline itoa, static memory buffers) manually or strictly use the host imports in <milf.h>.

Core constraints:
1. Include <milf.h>. Entrypoint: MILF_EXPORT int wasm_main(char* payload, int payload_len, char* out_buf, int out_max).
2. No stdlib headers/functions. Use milf.h inline utilities: milf_strlen, milf_memcpy, milf_memset, milf_memcmp, milf_abs, milf_min, milf_max.
3. Host imports in milf.h:
   - int milf_stream_open(const char* url) (returns handle > 0 or error < 0)
   - int milf_stream_read(int handle, char* target_buf, int chunk_size) (returns bytes or error < 0)
   - void milf_stream_close(int handle)
   - int milf_pdf_generate(const char* text, char* target_buf, int max_len) (returns size or error < 0)
   - int milf_storage_save(const char* name, const char* data, int len) (returns 0 or error < 0)
4. CRITICAL: DO NOT declare large static/global arrays (e.g., char buf[1024*1024*2]) for file/PDF generation in WASM memory, as it exceeds memory boundaries. Instead, reuse the pre-allocated out_buf (size out_max) as scratch space for milf_pdf_generate, save it with milf_storage_save, and then overwrite out_buf with the "FILE:out.pdf" string reference. Return the reference string length.
5. Payloads are wrapped as JSON (e.g. {"type":"json","data":"value"}). Write simple helper loops with milf_memcmp to search and extract values from the payload.

LEARNING CONTEXT (Pattern Examples):
1. Simple Add: MILF_EXPORT int wasm_main(int a, int b) { return a + b; }
2. Complex Data/JSON: Construct JSON manually using milf_memcpy. If payload empty, return "{\"error\": \"Empty payload\"}".
3. Network Fetch (milf_stream_*): 
   int handle = milf_stream_open("https://...");
   int bytes = milf_stream_read(handle, buf, size);
   milf_stream_close(handle);
4. Constructing Files/PDFs (Try2pdf/Imgtopdf):
   // Use out_buf directly as the scratchpad for generation to avoid massive static array allocations
   int pdf_size = milf_pdf_generate(text_buf, out_buf, out_max);
   milf_storage_save("out.pdf", out_buf, pdf_size);
   const char* ref = "FILE:out.pdf";
   milf_memcpy(out_buf, ref, milf_strlen(ref));
   return milf_strlen(ref);

Respond with a JSON object: {"insight": "markdown explanation of changes", "code": "complete compilable C code"}`

	userText := fmt.Sprintf("Prompt: %s", req.Prompt)
	if req.Code != "" {
		userText = fmt.Sprintf("%s\n\nCode to adapt/fix:\n```c\n%s\n```", userText, req.Code)
	}

	geminiReq := GeminiRequest{
		Contents: []GeminiContent{
			{
				Parts: []GeminiContentPart{
					{Text: userText},
				},
			},
		},
		SystemInstruction: GeminiSystemInstruction{
			Parts: []GeminiContentPart{
				{Text: systemText},
			},
		},
		GenerationConfig: GeminiGenerationConfig{
			ResponseMimeType: "application/json",
		},
	}

	reqBytes, err := json.Marshal(geminiReq)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to marshal Gemini request", err.Error())
		return
	}

	models := []string{"gemini-2.5-flash", "gemini-2.5-flash-lite"}
	var respBytes []byte
	var lastErr error
	var success bool

	for _, model := range models {
		url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", model, apiKey)
		httpReq, err := http.NewRequestWithContext(r.Context(), http.MethodPost, url, bytes.NewBuffer(reqBytes))
		if err != nil {
			lastErr = err
			continue
		}
		httpReq.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		httpResp, err := client.Do(httpReq)
		if err != nil {
			lastErr = err
			continue
		}

		respBytes, err = io.ReadAll(httpResp.Body)
		httpResp.Body.Close()
		if err != nil {
			lastErr = err
			continue
		}

		if httpResp.StatusCode == http.StatusOK {
			success = true
			break
		}
		lastErr = fmt.Errorf("model %s returned status %d: %s", model, httpResp.StatusCode, string(respBytes))
	}

	if !success {
		writeError(w, http.StatusBadGateway, "Failed to contact Gemini API (all models exhausted)", lastErr.Error())
		return
	}

	var geminiResp GeminiResponse
	if err := json.Unmarshal(respBytes, &geminiResp); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to parse Gemini response JSON", err.Error())
		return
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		writeError(w, http.StatusInternalServerError, "Empty response from Gemini API", "")
		return
	}

	rawText := geminiResp.Candidates[0].Content.Parts[0].Text

	var copilotResp CopilotResponse
	if err := json.Unmarshal([]byte(rawText), &copilotResp); err != nil {
		cleaned := rawText
		if start := bytes.Index([]byte(rawText), []byte("{")); start >= 0 {
			if end := bytes.LastIndex([]byte(rawText), []byte("}")); end >= 0 && end > start {
				cleaned = rawText[start : end+1]
			}
		}
		if err := json.Unmarshal([]byte(cleaned), &copilotResp); err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to parse Gemini response text as Copilot JSON", err.Error()+"\nRaw text: "+rawText)
			return
		}
	}

	writeJSON(w, http.StatusOK, copilotResp)
}

// HandleTestCopilotKey validates that a Gemini API Key is working
func (h *LambdaHandler) HandleTestCopilotKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed", "")
		return
	}

	apiKey := r.Header.Get("X-Gemini-Api-Key")
	if apiKey == "" {
		writeError(w, http.StatusBadRequest, "No API key provided", "X-Gemini-Api-Key header is missing")
		return
	}

	// Make an ultra-low-token request to verify the key works
	testReq := GeminiRequest{
		Contents: []GeminiContent{
			{
				Parts: []GeminiContentPart{
					{Text: "ping"},
				},
			},
		},
	}

	reqBytes, err := json.Marshal(testReq)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to marshal test request", err.Error())
		return
	}

	models := []string{"gemini-2.5-flash", "gemini-2.5-flash-lite"}
	var respBytes []byte
	var lastErr error
	var success bool

	for _, model := range models {
		url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", model, apiKey)
		httpReq, err := http.NewRequestWithContext(r.Context(), http.MethodPost, url, bytes.NewBuffer(reqBytes))
		if err != nil {
			lastErr = err
			continue
		}
		httpReq.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		httpResp, err := client.Do(httpReq)
		if err != nil {
			lastErr = err
			continue
		}

		respBytes, err = io.ReadAll(httpResp.Body)
		httpResp.Body.Close()
		if err != nil {
			lastErr = err
			continue
		}

		if httpResp.StatusCode == http.StatusOK {
			success = true
			break
		}
		lastErr = fmt.Errorf("model %s returned status %d: %s", model, httpResp.StatusCode, string(respBytes))
	}

	if !success {
		writeError(w, http.StatusUnauthorized, "Gemini API rejected key (all models exhausted)", lastErr.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"valid": true,
	})
}
