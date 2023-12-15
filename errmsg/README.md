# errmsg

Add end-user messages to errors.

`err.Error()` doesn't usually provide a human-friendly output. `errmsg` allows
errors to carry an (extendable) end-user message that can be used in e.g.
handlers.

Here is an example on how it can be used:

```go
package connector

import (
    // ...
    "github.com/instill-ai/x/errmsg"
)

func (c *Client) sendReq(reqURL, method, contentType string, data io.Reader) ([]byte, error) {
    // ...

    res, err := c.HTTPClient.Do(req)
    if err != nil {
        err := fmt.Errorf("failed to call connector vendor: %w", err)
        return nil, errmsg.AddMessage(err, "Failed to call Vendor API.")
    }

    if res.StatusCode < 200 || res.StatusCode >= 300 {
        err := fmt.Errorf("vendor responded with status code %d", res.StatusCode)
        msg := fmt.Sprintf("Vendor responded with a %d status code.", res.StatusCode)
        return nil, errmsg.AddMessage(err, msg)
    }

    // ...
}
```

```go
package handler

func (h *PublicHandler) DoAction(ctx context.Context, req *pb.DoActionRequest) (*pb.DoActionResponse, error) {
    resp, err := h.triggerActionSteps(ctx, req)
    if err != nil {
    resp.Outputs, resp.Metadata, err = h.triggerNamespacePipeline(ctx, req)
        return nil, status.Error(asGRPCStatus(err), errmsg.MessageOrErr(err))
    }

    return resp, nil
}
```
