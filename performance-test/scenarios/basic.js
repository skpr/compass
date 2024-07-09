import http from "k6/http"
import { check } from "k6"
    
export default function () {
  const baseURL = `http://127.0.0.1:8080/`
    
  let cb = (Math.random() + 1).toString(36).substring(7);
  let res = http.get(http.url`${baseURL}/en/articles?cache-buster=${cb}`)
  if (
    !check(res, {
      'ok': (res) => res.status == 200,
    })
  ) {
    fail('status code was *not* 200');
  }
}
