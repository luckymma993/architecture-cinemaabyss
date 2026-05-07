# To-Be Архитектура «Кинобездна» (C4 Container Diagram)

```plantuml
@startuml
!include https://raw.githubusercontent.com/plantuml-stdlib/C4-PlantUML/master/C4_Container.puml

Person(user, "User", "Пользователь (Web, Mobile, Smart TV)")

System_Ext(recsys, "Recommendation System", "Внешняя система, которая формирует персональные подборки фильмов.")

System_Boundary(c1, "CinemaAbyss") {
    Container(proxy, "API Gateway / Proxy", "Go", "Единая точка входа для всех клиентов. Плавно переключает трафик с монолита на новые сервисы без простоев.")
    
    Container(monolith, "Core System (Monolith)", "Go", "Наш старый монолит. Пока что продолжает отвечать за пользователей, платежи и подписки.")
    Container(movies, "Movies Service", "Go", "Новый микросервис, выделенный специально для работы с каталогом фильмов (жанры, актеры, оценки).")
    Container(events, "Events Service", "Go", "Сервис-прослойка для работы с событиями. Принимает информацию о действиях и отправляет её в Kafka.")
    
    ContainerDb(db, "PostgreSQL", "Relational Database", "Основная база данных. Временно осталась общей для монолита и новых сервисов ради плавного переезда.")
    ContainerQueue(kafka, "Apache Kafka", "Message Broker", "Шина сообщений. Позволяет сервисам общаться между собой асинхронно, не блокируя друг друга.")
}

Rel(user, proxy, "API Requests", "JSON/HTTPS")

' API Gateway маршрутизирует трафик
Rel(proxy, monolith, "Routes /api/users, /api/payments, /api/subscriptions", "REST/HTTP")
Rel(proxy, movies, "Routes /api/movies", "REST/HTTP")
Rel(proxy, events, "Routes /api/events", "REST/HTTP")

' Работа с БД
Rel(monolith, db, "Reads/Writes", "SQL/TCP")
Rel(movies, db, "Reads/Writes", "SQL/TCP")

' Работа с брокером сообщений
Rel(events, kafka, "Publishes & Consumes events", "TCP")

' Сторонняя система рекомендаций
Rel(kafka, recsys, "Delivers events (Async)", "TCP")
Rel(recsys, movies, "Provides recommendations", "REST/Async")
@enduml
```