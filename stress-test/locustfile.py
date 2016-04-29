from locust import HttpLocust, TaskSet, task


class FrodoTaskSet(TaskSet):

    @task(2)
    def ch_1(self):
        with self.client.get('/test-channel/1', catch_response=True) as r:
            r.success()

    @task(1)
    def ch_2(self):
        with self.client.get('/test-channel/2', catch_response=True) as r:
            r.success()

    @task(4)
    def ch_3(self):
        with self.client.get('/test-channel/3', catch_response=True) as r:
            r.success()

    @task(1)
    def ch_4(self):
        with self.client.get('/test-channel/4', catch_response=True) as r:
            r.success()


class FrodoLocust(HttpLocust):
    task_set = FrodoTaskSet
