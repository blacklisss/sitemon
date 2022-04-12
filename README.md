# Мониторинг сайтов
Отслеживание состояния сайтов с оповещением в Телеграм

## Настройка
Настройки задаются в файлах типа .yaml или .json

Пример yaml смотри в sitemom/cmd/server/config.example.yaml

По умолчанию исполуется файл с именем config.yaml
Поведение по умолчанию можно изменить с помощью флага config
go run ./... config="config.json"

## Поведение
Программа опрашивает список серверов раз в 30 минут. Если сервер не отвечает или статус ответа отличается от 200, 
то программа отправляет сообщение в Телеграм.

Чтобы все работало, необходимо зарегистрировать бота и узнать ID чата

### TODO
- [ ] Добавить в конфигурацию время опроса сервера
- [ ] Сделать докер
