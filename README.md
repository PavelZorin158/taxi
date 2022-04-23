Проект является Web приложением, предназначенным для контроля заказов и анализа дохода в такси

Проект находится в развитии

запускается на 5005 порте

собираем имидж с контейнером 
docker build .

docker images

переименовываем
docker tag 968da7c3b663 rick148/taxi:1.5.0

docker login
docker push rick148/taxi:1.5.0


остановить старый контейнер на сервере
docker stop 7a86cf430fb1

запуск контейнера на сервере
docker run -v rick_db:/usr/src/app/dir_db -p 5005:5005 rick148/taxi:1.5.0


