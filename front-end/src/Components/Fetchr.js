export default function fetchr(url, method, headers, body, handler, user, setUser) {
    let req = {method, headers, credentials: "include"}
    if(body != null) req.body = body
    if(user.accessToken != null)req.headers.Authorization = "Bearer " + user.accessToken
    fetch(url,req).then(response1 => {
        if (response1.status == 401) {
            console.log("Need to refresh tokens")
            //Need to refresh tokens after first fail
            fetch("http://localhost:8080/api/auth/refresh-token", {
                method: "POST",
                credentials: "include"
            }).then(response2 => {
                if (response2.status == 200) {
                    response2.json().then(json => {
                        window.localStorage.setItem("user", json.accessToken)
                        setUser({...user, accessToken: json.accessToken})
                        fetch(url, {
                            method,
                            headers: {
                                ...headers,
                                "Authorization": "Bearer " + json.accessToken
                            }, 
                            body
                        }).then(response3 => handler(response3)).catch(err => console.log(err))
                    })
                } else {
                    console.log("Token refreshing failed")
                    window.localStorage.removeItem("accessToken")
                    setUser({...user, accessToken: null})
                    throw new Error("request failed, code: ", response2.status)
                }
            }).catch(err => console.log(err))
        }else handler(response1)
    }).catch(err => {console.log(err)})
}
