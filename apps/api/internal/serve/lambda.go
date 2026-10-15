package serve

import (
	"net"
	"os"
	"net/http"
	"strings"

	"github.com/akrylysov/algnhsa"
	"github.com/aws/aws-lambda-go/lambda"
)

// OnLambda reports whether this process is an AWS Lambda function (Lambda sets the variable).
func OnLambda() bool { return os.Getenv("AWS_LAMBDA_FUNCTION_NAME") != "" }

// Lambda turns the router into a Lambda handler for API Gateway's HTTP API (payload v2),
// using algnhsa: the browser's cookies arrive as a Cookie header and every Set-Cookie goes back.
// hops is config.ForwardedHops: see clientAddr.
func Lambda(h http.Handler, hops int) lambda.Handler {
	return algnhsa.New(clientAddr(h, hops), &algnhsa.Options{RequestType: algnhsa.RequestTypeAPIGatewayV2})
}

// clientAddr sets r.RemoteAddr to the browser's address, which the patient sign-in rate limit
// counts by (httpapi clientIP). On AWS the connection comes from the last proxy, so the
// browser's address is read from X-Forwarded-For, where each of the hops trusted proxies has
// added the address it got the request from: the browser's is the hops-th from the right.
// Anything to its left was sent by the browser itself and is ignored. If the header has fewer
// addresses (the request didn't come through the proxies), the connection's address is kept.
func clientAddr(h http.Handler, hops int) http.Handler {
	if hops == 0 {
		return h
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var addrs []string
		for _, v := range r.Header.Values("X-Forwarded-For") {
			for _, a := range strings.Split(v, ",") {
				addrs = append(addrs, strings.TrimSpace(a))
			}
		}
		if len(addrs) >= hops {
			if ip := net.ParseIP(addrs[len(addrs)-hops]); ip != nil {
				r.RemoteAddr = ip.String()
			}
		}
		h.ServeHTTP(w, r)
	})
}
