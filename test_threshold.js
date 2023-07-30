const superagent = require('superagent');
const async = require('async');

const EVENTS = 50;
const PARALLEL_LIMIT = 30;
const INTERVAL_MS = 400;

const url = 'http://localhost:8090/report';

let body = {
    url: 'http://sample.com/abc',
};

function send_request(cb) {
    let parallels = [];
    let successful_responses = 0;

    const startTime = new Date();
    console.log(new Date().toISOString());

    body = Object.assign(
        {},
        { url: 'http://sample.com/abc' + Math.round(Math.random() * 1000) }
    );

    for (let i = 0; i < EVENTS; i++) {
        parallels.push((cb) => {
            superagent
                .post(url)

                .send(body)
                .then((res) => {
                    successful_responses += 1;
                    cb();
                })
                .catch((err) => {
                    if (err) console.error(err);
                });
        });
    }

    async.parallelLimit(parallels, PARALLEL_LIMIT, (err) => {
        if (err) return cb(err);
        console.log(new Date().toISOString());

        const endTime = new Date();
        const diff = endTime - startTime; //ms
        const seconds = diff / 1000;
        console.log('seconds: ', seconds, ' milliseconds: ', diff);
        console.log('Requests per second: ', successful_responses / seconds);
        console.log('successful_responses: ', successful_responses);
    });
}

setInterval(() => {
    send_request((err) => {
        if (err) console.error(err);
    });
}, INTERVAL_MS);
