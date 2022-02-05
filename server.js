const express = require("express");
const net = require("net");
const bitcoin = require("bitcoinjs-lib");
const BigNumber = require("bignumber.js");
const request = require("request");

//var electrsHost = 'electrs';
const btcRpcPort = process.env.BITCOIN_RPC_PORT;
const btcIp = process.env.BITCOIN_IP;
const btcPassword = process.env.BITCOIN_RPC_PASS;
const btcRpcUrl = `${btcIp}:${btcRpcPort}/`;
const electrsHost = process.env.ELECTRUM_IP;
const electrsPort = process.env.ELECTRUM_PORT;
const btcRpcAuth = `umbrel:${btcPassword}`;
// console.log(`INFO PROXY: btc rpc pass: ${btcCookiePass}`)
console.log(`Bitcoin IP: ${btcIp}`);
console.log(`Bitcoin port: ${btcRpcPort}`);
console.log(`Bitcoin URL: ${btcRpcUrl}`);
console.log(`Electrs IP: ${electrsHost}`);
console.log(`Electrs port: ${electrsPort}`);
console.log(`Bitcoin RPC auth: ${btcRpcAuth}`);
console.log(`INFO PROXY: Electrs host: ${electrsHost}:${electrsPort}`);

const network =
  process.env.BTC_NETWORK == "TESTNET"
    ? bitcoin.networks.testnet
    : bitcoin.networks.bitcoin;

const app = express();
const port = 50002;
app.use(express.json());

const identity = (x) => x;

const addressToScriptHash = (address) => {
  let script = bitcoin.address.toOutputScript(address, network);
  let hash = bitcoin.crypto.sha256(script);
  let reversedHash = Buffer.from(hash.reverse());
  return reversedHash.toString("hex");
};

/* takes BTC amount, returns Satoshis */
const toSats = (btc) => {
  const sats = new BigNumber(btc).times(100000000).toNumber();
  return Math.ceil(sats);
};

// electrs rpc
const eRpc = (rpcCall, addr) => {
  return new Promise((resolve, reject) => {
    let scriptHash;
    let params = {};

    if (addr !== undefined) {
      try {
        scriptHash = addressToScriptHash(addr);
      } catch (e) {
        return reject({ code: 400, msg: "bad address to electrs-rpc" });
      }
      params = { params: [scriptHash] };
    }

    const client = new net.Socket();
    client.connect(electrsPort, electrsHost, () => {
      const rc = Object.assign(params, rpcCall);
      client.write(JSON.stringify(rc));
      client.write("\r\n");
    });
    client.on("error", (err) => {
      return reject({ code: 502, msg: "e-rpc error" });
    });
    client.on("data", (data) => {
      client.destroy();
      resolve(JSON.parse(data.toString()));
    });
  });
};

// btc rpc
const bRpc = (rpcCall) => {
  return new Promise((resolve, reject) => {
    const headers = {
      "content-type": "text/plain;",
    };
    const options = {
      url: `http://${btcRpcAuth}@${btcRpcUrl}`,
      method: "POST",
      headers: headers,
      body: JSON.stringify(rpcCall),
    };
    const callback = (error, response, body) => {
      if (!error && response.statusCode == 200) {
        return resolve(JSON.parse(body));
      } else {
        let err;
        try {
          err = JSON.parse(body).error;
        } catch (e) {
          return reject({ code: 400, msg: "bad btc-rpc call" });
        }
        if (err != undefined) {
          return resolve(JSON.parse(body));
        } else {
          return reject({ code: 400, msg: "bad btc-rpc call" });
        }
      }
    };
    try {
      request(options, callback);
    } catch (e) {
      return reject({ code: 502 });
    }
  });
};

// takes a promise, transforms its JSON result, and responds
const jsonRespond = (rpcPromise, transformer, res) => {
  rpcPromise
    .then((json) => {
      res.send(transformer(json));
    })
    .catch((err) => {
      console.log(err);
      res.status(err.code).end();
    });
};

/*
  Composes 3 separate RPC calls to:
    - electrs: listunspent
    - electrs: get_history
    - btc:     getblockcount,
  and packages the results into one RPC return
*/
app.get("/addresses/info/:address", (req, res) => {
  const address = req.params.address;
  const id = "get-address-info";
  const rpcCall1 = {
    jsonrpc: "2.0",
    id,
    method: "blockchain.scripthash.listunspent",
  };
  const rpcCall2 = {
    jsonrpc: "2.0",
    id: "e-rpc",
    method: "blockchain.scripthash.get_history",
  };
  const blockRpc = { jsonrpc: "2.0", id: "btc-rpc", method: "getblockcount" };
  const timeRpc = { jsonrpc: "2.0", id: "btc-rpc", method: "getblockstats" };

  let utxos;
  let used;
  let block;
  eRpc(rpcCall1, address)
    .then((json) => {
      utxos = json.result;
      return eRpc(rpcCall2, address);
    })
    .then((json) => {
      used = utxos.length > 0 || json.result.length > 0;
      return bRpc(blockRpc);
    })
    .then((json) => {
      block = json.result;
      const ps = utxos.map((u) => {
        if (u.height == 0) {
          return { result: { time: 0 } };
        } else {
          const params = [u.height, ["time"]];
          const call = { ...timeRpc, params };
          return bRpc(call);
        }
      });
      return Promise.all(ps);
    })
    .then((jsons) => {
      for (let i = 0; i < jsons.length; i++) {
        utxos[i] = { ...utxos[i], recvd: jsons[i].result.time };
      }
      res.send({ error: null, id, result: { address, utxos, block, used } });
    })
    .catch((err) => {
      console.log(err);
      res.status(500).end();
    });
});

app.get("/getblockinfo/:block?", (req, res) => {
  let block = Number(req.params.block);
  let fee;
  let blockhash;
  const id = "get-block-info";

  new Promise((resolve, reject) => {
    if (!req.params.block) {
      const blockCall = {
        jsonrpc: "2.0",
        id: "btc-rpc",
        method: "getblockcount",
      };
      bRpc(blockCall).then((json) => {
        return resolve({ block: json.result });
      });
    } else if (Number.isInteger(block) && block >= 0) {
      return resolve({ block: block });
    } else {
      return reject({
        code: 400,
        msg: `invalid block parameter: ${req.params.block}`,
      });
    }
  })
    .then((json) => {
      block = json.block;
      const feeCall = {
        jsonrpc: "2.0",
        id: "btc-rpc",
        method: "estimatesmartfee",
        params: [1],
      };
      return bRpc(feeCall);
    })
    .then((json) => {
      if (json.result && !json.result.errors) {
        // fee is per kilobyte, we want in bytes
        fee = Math.ceil(toSats(json.result.feerate) / 1024);
      } else {
        fee = null;
      }
      const blockhashCall = {
        jsonrpc: "2.0",
        id: "btc-rpc",
        method: "getblockhash",
        params: [block],
      };
      return bRpc(blockhashCall);
    })
    .then((json) => {
      blockhash = json.result;
      const blockfilterCall = {
        jsonrpc: "2.0",
        id,
        method: "getblockfilter",
        params: [blockhash],
      };
      return bRpc(blockfilterCall);
    })
    .then((json) => {
      res.send({
        ...json,
        result: { block, fee, blockhash, blockfilter: json.result.filter },
      });
    })
    .catch((err) => {
      console.log(err);
      res.status(err.code).end();
    });
});

app.get("/getblockcount", (req, res) => {
  const id = "get-block-count";
  const rpcCall = { jsonrpc: "2.0", id, method: "getblockcount" };
  jsonRespond(bRpc(rpcCall), identity, res);
});

app.get("/getrawtx/:txid", (req, res) => {
  const id = "get-raw-tx";
  const txid = req.params.txid;
  const rpcCall = {
    jsonrpc: "2.0",
    id,
    method: "getrawtransaction",
    params: [txid],
  };
  const addTxId = (json) => {
    return { ...json, result: { rawtx: json.result, txid } };
  };
  jsonRespond(bRpc(rpcCall), addTxId, res);
});

app.get("/gettxvals/:txid", (req, res) => {
  const id = "get-tx-vals";
  const txid = req.params.txid.toLowerCase();
  const rpcCall = {
    jsonrpc: "2.0",
    id,
    method: "getrawtransaction",
    params: [txid, true],
  };
  let vouts;
  let included;
  let outputs;
  let confs;
  let recvd;
  bRpc(rpcCall)
    .then((json) => {
      //  If error is -5, TX not in blockchain
      if (json.error != null && json.error.code === -5) {
        included = false;
        confs = 0;
        recvd = 0;
        outputs = [];
        return Promise.all([]);
      }

      included = true;
      confs = json.result.confirmations;
      recvd = json.result.blocktime;

      outputs = json.result.vout.map((vout) => {
        return {
          txid,
          address: vout.scriptPubKey.address,
          pos: vout.n,
          value: toSats(vout.value),
        };
      });
      vouts = json.result.vin.map((vin) => vin.vout);
      const tmpInputs = json.result.vin.map((vin) => {
        const call = {
          jsonrpc: "2.0",
          id: "tmp",
          method: "getrawtransaction",
          params: [vin.txid, true],
        };
        return bRpc(call);
      });
      return Promise.all(tmpInputs);
    })
    .then((jsons) => {
      const inputs = jsons.map((j, idx) => {
        const vout = j.result.vout[vouts[idx]];
        return {
          txid: j.result.txid,
          address: vout.scriptPubKey.address,
          pos: vout.n,
          value: toSats(vout.value),
        };
      });
      if (confs === undefined) confs = 0;
      if (recvd === undefined) recvd = 0;
      res.send({
        error: null,
        id,
        result: { included, txid, confs, recvd, inputs, outputs },
      });
    })
    .catch((err) => {
      console.log(err);
      res.status(err.code).end();
    });
});

app.get("/broadcasttx/:rawtx", (req, res) => {
  const id = "broadcast-tx";
  const txid = bitcoin.Transaction.fromHex(req.params.rawtx).getId();
  const sendTxCall = {
    jsonrpc: "2.0",
    id,
    method: "blockchain.transaction.broadcast",
    params: [req.params.rawtx],
  };
  const txInfoCall = {
    jsonrpc: "2.0",
    id,
    method: "blockchain.transaction.get",
    params: [txid, true],
  };

  eRpc(sendTxCall)
    .then((json) => {
      if (json.result != null) {
        // got txid, done
        res.send({
          ...json,
          result: { txid, broadcast: true, included: false },
        });
      } else {
        return eRpc(txInfoCall);
      }
    })
    // we only get here if sendrawtransaction failed
    .then((json) => {
      // -5 : getrawtransaction failed with unseen
      if (json.error != null && json.error.code === -5) {
        res.send({
          ...json,
          error: null,
          result: { txid, broadcast: false, included: false },
        });
      }
      // otherwise, we saw the transaction, but it failed to add, means it already succeeded
      else {
        res.send({
          ...json,
          result: { txid, broadcast: false, included: true },
        });
      }
    })
    .catch((err) => {
      console.log(err);
      res.status(err.code).end();
    });
});

app.listen(port, () => console.log(`Electrs proxy listening on port ${port}`));
// Modified server.js
