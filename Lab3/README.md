# Лабораторная работа 3 — Платформа для shop

## Часть 1. Guardrails для кластера (Kyverno)

Для использования в работе выбран движок **Kyverno**, потому что политики пишутся на **YAML**, а не на специальном языке.

```bash
helm repo add kyverno https://kyverno.github.io/kyverno/
helm repo update
helm install kyverno kyverno/kyverno -n kyverno --create-namespace
```
Проверка:

![alt text](Screenshots/image.png)
### Первая политика

```yaml
apiVersion: kyverno.io/v1
kind: ClusterPolicy
metadata:
  name: require-limits
  annotations:
    policies.kyverno.io/title: Require resource limits
    policies.kyverno.io/description: >-
      Каждый контейнер обязан указать limits по памяти и CPU.
      Без лимитов под может сожрать всю ноду и уронить соседей.
spec:
  validationFailureAction: Enforce   # Enforce = отклонять, Audit = только логировать
  background: true
  rules:
  - name: check-limits
    match:
      any:
      - resources:
          kinds:
          - Pod
    validate:
      message: "У контейнера должны быть заданы resources.limits.memory и resources.limits.cpu"
      pattern:
        spec:
          containers:
          - resources:
              limits:
                memory: "?*"
                cpu: "?*"
```

Применяем и проверяем:

```bash
kubectl apply -f policies/require-limits.yaml
kubectl get clusterpolicy
```
![alt text](Screenshots/image-1.png)

### Пять политик и пять «плохих» манифестов

Созданы пять политик:

1. требование `resources.limits.memory` и `resources.limits.cpu` для контейнера;
2. запрет привилегированных контейнеров;
3. требование обязательных лейблов `app` и `version`;
4. образы только из `docker.io`, `gcr.io`, `ghcr.io`;
5. запрет запуска контейнеров от рута.

Также созданы пять плохих манифестов для их проверки.

Применяем политики:

![alt text](Screenshots/image-2.png)
![alt text](Screenshots/image-3.png)

Проверяем, отобьёт ли Kyverno плохие манифесты:

- **Лимиты:**

![alt text]Screenshots/(image-4.png)
- **Привилегированный контейнер:**

![alt text](Screenshots/image-5.png)
- **Отсутствие лейблов:**

![alt text](Screenshots/image-6.png)
- **Недоверенный источник:**

![alt text](Screenshots/image-7.png)
- **Контейнер-рут:**

![alt text](Screenshots/image-8.png)

Проверяем хороший под:

![alt text](Screenshots/image-9.png)

Все плохие манифесты были отбиты, а хороший принят.

### Снижение нагрузки от Kyverno

Т.к. Kyverno начал жрать процессор, во всех политиках поле `background` поставлено в `false` и отключены контроллеры, отвечающие за проверку существующих объектов и написание отчётов на каждое событие:

```bash
kubectl scale --replicas=0 deployment/kyverno-background-controller deployment/kyverno-reports-controller -n kyverno
kubectl delete ur,polr,cpolr,ephr,cephr --all -A
```

## Часть 2. Helm-чарт для api и worker

Собираем образы с нужными тегами:

```bash
docker build -t docker.io/library/shop-api:v2 ./api
docker build -t docker.io/library/shop-worker:v2 ./worker
```

Экспортируем из docker в containerd:

```bash
docker save docker.io/library/shop-api:v2 -o api.tar
docker save docker.io/library/shop-worker:v2 -o worker.tar
sudo /usr/local/bin/k3s ctr images import api.tar
sudo /usr/local/bin/k3s ctr images import worker.tar
```

Добавляем `api-service.yaml`. Устанавливаем чарт:

```bash
helm install shop ./charts/shop
```

Видим три пода `api` и два пода `worker`:

![alt text](Screenshots/image-10.png)

Также видим Service для `api`:

![alt text](Screenshots/image-11.png)

### Reconciliation

Удаляем поды `api`:

![alt text](Screenshots/image-12.png)

Видим, что все поды восстановились.


Меняем число реплик:

```bash
sed -i 's/replicas: 3/replicas: 5/' charts/shop/values.yaml
```

![alt text](Screenshots/image-13.png)

Видим, что количество реплик автоматически увеличилось до 5.


### Rolling update

Пробрасываем порт:

```bash
kubectl port-forward svc/shop-api 8080:80
```

И начинаем долбёжку в `/health`:

![alt text](Screenshots/image-14.png)

Поднимаем версию, апгрейдим и наблюдаем за curl:

![alt text](Screenshots/image-15.png)

Видим, что что-то пошло не так. Вероятно, проброс портов оказался мёртв вместе с убитым подом, т.к. автоматического переключения на новый под нет. Чтобы правильно проверить rolling update, запускаем генератор нагрузки как под в кластере:

![alt text](Screenshots/image-16.png)

Видим стабильный поток ответов 200. Снова поднимаем версию:

![alt text](Screenshots/image-17.png)
Видим:
![alt text](Screenshots/image-18.png)

Какие-то запросы провалились. Я сделал вывод, что натворил какую-то дичь, и начал заново.

Поднимаем прокси:

```bash
kubectl proxy
```

Цикл:

```bash
while true; do
  code=$(curl -s -o /dev/null -w "%{http_code}" \
    http://localhost:8001/api/v1/namespaces/default/services/shop-api:80/proxy/health)
  echo "$code"
  sleep 0.2
done
```

Поды:

![alt text](Screenshots/image-19.png)

Заново собираем и поднимаем версию:

![alt text](Screenshots/image-20.png)

Видим, что поды рестартнулись, фэйлов нет, все ответы 200.

Тут видно, что поды умирают и запускаются не одновременно:

![alt text](Screenshots/image-21.png)

Это говорит о том, что rolling update работает и в любой момент времени есть работающие поды.

### Сломанная версия и rollback

Выставляем api переменную `HEALTH_FAIL=true` и апгрейдим.

Видим:

![alt text](Screenshots/image-22.png)

Поды не обновляются, но сервис продолжает всё время отвечать, потому что из пяти старых подов убит только один.

Откатываем обратно:

![alt text](Screenshots/image-23.png)

Видим, что все пять подов на месте.

## Часть 3. Postgres через оператор CloudNativePG

Разворачиваем оператор БД **CloudNativePG**. Проверяем работу:

![alt text](Screenshots/image-24.png)

Пробуем удалить под postgres и проверить, вернёт ли его оператор. Список текущих подов:

![alt text](Screenshots/image-25.png)

Удаляем под postgres:

![alt text](Screenshots/image-26.png)

Видим, что оператор его вернул.


### Чем оператор отличается от controller-manager

**kube-controller-manager** — это набор встроенных контроллеров Kubernetes (ReplicaSet, Deployment, StatefulSet, Job). Они понимают только стандартные ресурсы API и обеспечивают базовую reconciliation-логику: «должно быть N реплик — держи N реплик». Если под умер, ReplicaSet создаст новый. Но controller-manager не знает специфики приложений: он не умеет делать backup PostgreSQL, не умеет переключать primary при failover, не умеет управлять репликацией или миграциями схемы.

**Оператор (CloudNativePG)** — это специализированный контроллер, который понимает кастомный ресурс `Cluster` (CRD) и знает про PostgreSQL всё. Из одного объекта `Cluster` он создаёт и поддерживает:

1. поды инстансов (StatefulSet под капотом);
2. PVC для данных;
3. сервисы (`-rw` для записи, `-ro` для чтения, `-r` для реплик);
4. секреты с CA и TLS-сертификатами;
5. PodDisruptionBudget;
6. репликацию, failover, backup.

Оператор постоянно смотрит на `spec` (желаемое состояние) и `status` (наблюдаемое состояние) и гонит фактическое к желаемому. Если primary упал — оператор сам выберет новый primary из реплик (если их больше одной), обновит `currentPrimary` в status и переключит трафик на `-rw` сервис. Controller-manager так не умеет — он просто пересоздаст упавший под, но не знает про роли в PostgreSQL.

**Главное отличие:** controller-manager универсальный и работает с любыми подами одинаково. Оператор — специализированный, знает предметную область конкретного приложения и автоматизирует операционную рутину (backup, failover, scaling), которую иначе делал бы инженер вручную.

## Часть 4. Отказ control plane

Останавливаем k3s, чтобы доказать, что control plane работать не будет, но при этом сервис продолжит работать.

Сначала убеждаемся, что всё работает до эксперимента:

![alt text](Screenshots/image-27.png)

Включаем NodePort:

```bash
kubectl patch svc shop-api -p '{"spec":{"type":"NodePort"}}'
```

![alt text](Screenshots/image-28.png)

Останавливаем k3s:

```bash
sudo systemctl stop k3s
```

Видим, что control plane не работает (`kubectl get pods` → connection refused), а сервис отвечает (`curl` на NodePort → ok).

Возвращаем control plane:

```bash
sudo systemctl start k3s
```

![alt text](Screenshots/image-29.png)

Видим, что всё снова работает.

## Часть 5. Мониторинг

Добавим мониторинг в эту лабу, используя стек из лабы 2 (часть стека – логи и трейсы не нужны).
Чтобы показать хотя бы одно сработавшее правило, добавим stateful postgres, который не поднимется, и увидим:

![alt text](Screenshots/image-30.png)

Промитиус зафиксировал неработоспособность пода.
Собираем дашборд (метрики: рестарты, состояние постгреса и доступные реплики api):

![alt text](Screenshots/image-31.png)

Видим графики для метрик 
