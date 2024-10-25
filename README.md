# Сервис для записи прямых трансляций на площадках Twitch/YouTube/Kick и других
Проект представляет собой сервис для записи прямых трансляций на площадках Twitch/YouTube/Kick и других, управляемый через API.

# Технологический стек
## Backend
- Golang
- REST API (фреймворк Gin)

## Документация
Пользовательскую документацию можно получить по данной [ссылке](https://ra1phdd.github.io/stream-recorder/).

# Установка и запуск
- Клонируйте репозиторий:
```
git clone [https://github.com/ra1phdd/GetYTStatsAPI](https://github.com/ra1phdd/stream-recorder.git
```
- Перейдите в директорию проекта:
```
cd stream-recorder
```
- Установите зависимости для Backend:
```
go mod download
```
- Создайте файл .env со следующими параметрами:
```
PORT=8080
LOGGER_LEVEL=warn
GIN_MODE=release
ROOT_PATH=YOUR_ROOT_PATH
SPLIT_SEGMENTS=true
TIME_SEGMENT=1200
TIME_CHECK=15
VIDEO_CODEC=YOUR_VIDEO_CODEC
AUDIO_CODEC=YOUR_AUDIO_CODEC
FILE_FORMAT=YOUR_FILE_FORMAT
```
**Примечание**: Значение YOUR_ROOT_PATH являются обязательным.

- Замените YOUR_VIDEO_CODEC и YOUR_AUDIO_CODEC, YOUR_FILE_FORMAT на соответствующие значения, исходя из документации ffmpeg.
- Запустите Backend:
```
go run ./cmd/main/main.go
```

# Лицензия
Этот проект лицензируется под лицензией MIT. Подробнее см. [LICENSE](https://github.com/ra1phdd/stream-recorder/blob/master/LICENSE).
