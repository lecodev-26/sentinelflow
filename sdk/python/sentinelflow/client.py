from __future__ import annotations
import json
from dataclasses import dataclass
from typing import Any
from urllib.request import Request, urlopen
from urllib.error import HTTPError

@dataclass
class SFError(Exception):
    status: int
    type: str
    message: str
    def __str__(self): return f"sentinelflow: {self.status} {self.type}: {self.message}"

class Client:
    def __init__(self, base_url: str, api_key: str, timeout: float = 30):
        self.base_url=base_url.rstrip("/")
        self.api_key=api_key
        self.timeout=timeout
    def _request(self, method: str, path: str, body: dict[str,Any]|None=None):
        data=json.dumps(body).encode() if body is not None else None
        req=Request(self.base_url+path,data=data,method=method,headers={"Authorization":f"Bearer {self.api_key}","Content-Type":"application/json"})
        try:
            with urlopen(req,timeout=self.timeout) as r: return json.loads(r.read())
        except HTTPError as e:
            raw=e.read()
            try: err=json.loads(raw).get("error",{})
            except Exception: err={}
            raise SFError(e.code,err.get("type","http_error"),err.get("message",raw.decode(errors="replace")))
    def chat(self, model: str, messages: list[dict[str,str]], stream: bool=False):
        return self._request("POST","/v1/chat/completions",{"model":model,"messages":messages,"stream":stream})
    def responses(self, model: str, input: str):
        return self._request("POST","/v1/responses",{"model":model,"input":input})
    def models(self):
        return self._request("GET","/v1/models")["data"]
