# go-musthave-diploma-tpl

![alt text](http://github.com/Alena-Kurushkina/gophermart.git/db_erd.png)

# Диаграммы решения

```mermaid
sequenceDiagram
    participant Alice
    participant Bob
    Alice->>John: Hello John, how are you?
    loop HealthCheck
        John->>John: Fight against hypochondria
    end
    Note right of John: Rational thoughts <br/>prevail!
    John-->>Alice: Great!
    John->>Bob: How about you?
    Bob-->>John: Jolly good!
```

### Алгоритмы обработчиков

#### **Регистрация пользователя**

* `POST /api/user/register` 

- Без мидлваря аутентификации
- Каждый логин должен быть уникальным.
- После успешной регистрации должна происходить автоматическая аутентификация пользователя.

Формат запроса:

```
POST /api/user/register HTTP/1.1
Content-Type: application/json
...

{
	"login": "<login>",
	"password": "<password>"
}
```

Возможные коды ответа:

- `200` — пользователь успешно зарегистрирован и аутентифицирован;
- `400` — неверный формат запроса;
- `409` — логин уже занят;
- `500` — внутренняя ошибка сервера.

1. Проверка Content-Type
2. Получение тела запроса
3. Сокрытие пароля(хэш с солью)
4. Генерация uuid
4. Отправка запроса в БД для добавления записи (логин и пароль) в таблицу users.
    Если возникла ошибка (логин не уникальный), то возвращаем 409
5. Формируем JWT и кладём её в куки
6. Возвращаем 200

#### **Аутентификация пользователя**

Хендлер: `POST /api/user/login`.

- Без мидлваря аутентификации
- Аутентификация производится по паре логин/пароль.

Формат запроса:

```
POST /api/user/login HTTP/1.1
Content-Type: application/json
...

{
	"login": "<login>",
	"password": "<password>"
}
```

Возможные коды ответа:

- `200` — пользователь успешно аутентифицирован;
- `400` — неверный формат запроса;
- `401` — неверная пара логин/пароль;
- `500` — внутренняя ошибка сервера.

1. Проверка Content-Type
2. Получение тела запроса
3. Сокрытие пароля
4. Отправка запроса в БД дял проверки совпадения пароля для указанного логина (select id from users where login like login and password like password)
    Если не совпадает, то возвращаем 401
5. Формируем JWT и клалём в куки
6. Возвращаем 200