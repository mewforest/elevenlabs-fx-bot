package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"
	"net/http"
	"os"
	"strings"
)

type GenerationSettings struct {
	DurationSeconds *int    `json:"duration_seconds"`
	PromptInfluence float64 `json:"prompt_influence"`
}

type GenerationConfig struct {
	Text                string             `json:"text"`
	GenerationSettings  GenerationSettings `json:"generation_settings"`
	NumberOfGenerations int                `json:"number_of_generations"`
}

type SoundGenerationHistoryItem struct {
	SoundGenerationHistoryItemID string           `json:"sound_generation_history_item_id"`
	Text                         string           `json:"text"`
	CreatedAtUnix                int64            `json:"created_at_unix"`
	ContentType                  string           `json:"content_type"`
	GenerationConfig             GenerationConfig `json:"generation_config"`
}

type SoundGenerationWithWaveform struct {
	SoundGenerationHistoryItem SoundGenerationHistoryItem `json:"sound_generation_history_item"`
	WaveformBase64             string                     `json:"waveform_base_64"`
}

type TextToFxResponse struct {
	SoundGenerationsWithWaveforms []SoundGenerationWithWaveform `json:"sound_generations_with_waveforms"`
}

func generateSound(text string, promptInfluence float64, durationSeconds *int) (*TextToFxResponse, error) {
	log.Infof("Generating sound for '%v' (prompt influence: %v, duration: %v)...", text, promptInfluence, durationSeconds)
	headers := map[string]string{
		"accept":             "*/*",
		"accept-language":    "ru-RU,ru;q=0.9,en-US;q=0.8,en;q=0.7",
		"cache-control":      "no-cache",
		"content-type":       "application/json",
		"dnt":                "1",
		"origin":             "https://elevenlabs.io",
		"pragma":             "no-cache",
		"priority":           "u=1, i",
		"referer":            "https://elevenlabs.io/",
		"sec-ch-ua-mobile":   "?0",
		"sec-ch-ua-platform": "\"Windows\"",
		"sec-fetch-dest":     "empty",
		"sec-fetch-mode":     "cors",
		"sec-fetch-site":     "same-site",
		"user-agent":         "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36",
	}

	body := map[string]interface{}{
		"text":             text,
		"prompt_influence": promptInfluence,
		"duration_seconds": durationSeconds,
	}
	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal body: %v", err)
	}

	req, err := http.NewRequest("POST", "https://api.elevenlabs.io/sound-generation", bytes.NewBuffer(bodyJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	var result TextToFxResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}
	log.Infof("Sounds generated: %v", len(result.SoundGenerationsWithWaveforms))

	return &result, nil
}

func decodeResponseAudioFiles(response *TextToFxResponse) ([][]byte, error) {
	log.Infof("Decoding audio files...")
	audioBytes := make([][]byte, len(response.SoundGenerationsWithWaveforms))
	if len(response.SoundGenerationsWithWaveforms) == 0 {
		return audioBytes, nil
	}
	for i, item := range response.SoundGenerationsWithWaveforms {
		data, err := base64.StdEncoding.DecodeString(item.WaveformBase64)
		if err != nil {
			fmt.Printf("Error decoding base64: %v\n", err)
			return nil, err
		}
		audioBytes[i] = data
	}
	log.Infof("Decoded audio files: %v", len(audioBytes))
	// log also sizes of audios
	for i, audio := range audioBytes {
		log.Infof("[decoded] Audio %v size: %v", i, len(audio))
	}
	return audioBytes, nil
}

//func saveResponseFiles(response *TextToFxResponse) bool {
//	if len(response.SoundGenerationsWithWaveforms) == 0 {
//		return false
//	}
//
//	for _, item := range response.SoundGenerationsWithWaveforms {
//		fileName := item.SoundGenerationHistoryItem.SoundGenerationHistoryItemID + ".mp3"
//		data, err := base64.StdEncoding.DecodeString(item.WaveformBase64)
//		if err != nil {
//			fmt.Printf("Error decoding base64: %v\n", err)
//			return false
//		}
//		// Save to file with os library here
//		fmt.Printf("Saved to file: %s\n", fileName)
//
//		// Save to file with ioutil
//		if err := ioutil.WriteFile(fileName, data, 0644); err != nil {
//			fmt.Printf("Error writing file: %v\n", err)
//			return false
//		}
//	}
//
//	return true
//}

//func main() {
//	result, err := generateSound("kick", 0.3, nil)
//	if err != nil {
//		fmt.Printf("Error generating sound: %v\n", err)
//		return
//	}
//	if result == nil {
//		fmt.Println("No result")
//		return
//	}
//
//	saveResponseFiles(result)
//}

func actionsHandler(bot *tgbotapi.BotAPI, update tgbotapi.Update) error {
	// Check if the message is a command
	if update.Message.IsCommand() {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")
		msg.ParseMode = "HTML"
		msg.DisableWebPagePreview = true

		errorHandler := func(err error) {
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")
			msg.ParseMode = "HTML"
			msg.Text = fmt.Sprintf("Error generating sound: <code>%v</code>", err)
			_, _ = bot.Send(msg)
		}

		switch update.Message.Command() {
		case "start":
			msg.Text = "Привет! Я бот для генерации звуков c помощью AI.\n\n" +
				"Напиши /generate c текстом для генерации аудио из текста. Например: <code>/generate kick</code>\n\n" +
				"Бот использует сторонний сервис ElevenLabs, поэтому, пожалуйста, соблюдайте его правила и не злоупотребляйте функционалом бота.\n\n" +
				"Кстати, обязательно подписывайся на мой канал: @mewnotes ⭐"
			_, err := bot.Send(msg)
			if err != nil {
				errorHandler(err)
				return err
			}
		case "generate":
			// Check if the command has arguments
			if update.Message.CommandArguments() == "" {
				msg.Text = "🤔 Напиши текст для генерации аудио. Например: <code>/generate kick</code>"
				_, err := bot.Send(msg)
				if err != nil {
					errorHandler(err)
				}
				return nil
			}
			// Generating sound
			msg.Text = "🎵 Начинаю генерацию аудио..."
			_, err := bot.Send(msg)
			if err != nil {
				errorHandler(err)
				return err
			}

			// Generate sound
			result, err := generateSound(update.Message.CommandArguments(), 0.3, nil)
			if err != nil {
				msg.Text = fmt.Sprintf("Error generating sound: %v", err)
				_, err := bot.Send(msg)
				return err
			}
			if result == nil {
				err = fmt.Errorf("no result")
				errorHandler(err)
				return err
			}

			// Decode audio files
			audioBytes, err := decodeResponseAudioFiles(result)
			if err != nil {
				errorHandler(err)
				return err
			}

			// Send audio files
			if len(audioBytes) == 0 {
				msg.Text = "😔 Сервис вернул пустой результат, такое бывает. Попробуйте еще раз."
				_, err := bot.Send(msg)
				return err
			} else if len(audioBytes) > 1 {
				msg.Text = "🥁 Сервис сгенерировал несколько аудиофайлов, загружаю их по очереди..."
				_, err := bot.Send(msg)
				if err != nil {
					errorHandler(err)
					return err
				}
			}
			for _, audio := range audioBytes {
				file := tgbotapi.FileBytes{Name: "audio.mp3", Bytes: audio}
				audioMsg := tgbotapi.NewAudioUpload(update.Message.Chat.ID, file)
				_, err = bot.Send(audioMsg)
				if err != nil {
					errorHandler(err)
					return err
				}
			}

			msg.Text = fmt.Sprintf("🎵 Генерация аудио завершена! Чтобы сгенерировать еще, напиши <code>/%s %s</code>",
				update.Message.Command(), update.Message.CommandArguments())
			_, err = bot.Send(msg)
			if err != nil {
				errorHandler(err)
				return err
			}

			//// Testing ausio sending
			//// Assuming you have an audio file named "audio.mp3" in the same directory
			//audioBytes, err := os.ReadFile("audio.mp3")
			//if err != nil {
			//	log.Panic(err)
			//}
			//
			//file := tgbotapi.FileBytes{Name: "audio.mp3", Bytes: audioBytes}
			//audio := tgbotapi.NewAudioUpload(update.Message.Chat.ID, file)
			//
			//_, err = bot.Send(audio)
			//if err != nil {
			//	log.Panic(err)
			//}

		default:
			msg.Text = "I don't know that command"
			_, err := bot.Send(msg)
			return err
		}
	}
	return nil
}

func main() {
	// loading envs
	if err := godotenv.Load(); err != nil {
		log.Print("No .env file found")
	}

	telegramBotApiToken := os.Getenv("TELEGRAM_BOT_API_TOKEN")
	if telegramBotApiToken == "" {
		log.Fatal("TELEGRAM_BOT_API_TOKEN is not set in .env")
	}
	isBotDebug := strings.ToLower(os.Getenv("BOT_DEBUG")) == "true"

	// start telegram bot
	bot, err := tgbotapi.NewBotAPI(telegramBotApiToken)
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = isBotDebug

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil { // ignore any non-Message Updates
			continue
		}

		log.Printf("[@%s (id:%s)] %s", update.Message.From.UserName, update.Message.From.ID, update.Message.Text)

		// Handlers
		err := actionsHandler(bot, update)
		if err != nil {
			log.Printf("Error handling action: %v", err)
		}
	}
}
