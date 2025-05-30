// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {Script} from "forge-std/Script.sol";
import {console} from "forge-std/console.sol";

contract TestSimpleArtifactScript is Script {
    function run() public {
        console.log("Testing simple artifact deployment...");
        
        // Test deploying Counter first
        try vm.deployCode("Counter.sol:Counter") returns (address counter) {
            console.log("Counter SUCCESS! Deployed at:", counter);
        } catch Error(string memory reason) {
            console.log("Counter failed with:", reason);
        } catch {
            console.log("Counter failed with low-level error");
        }
        
        // Test deploying PrivateKeySender with proper args
        (address testAddr, uint256 testKey) = makeAddrAndKey("test");
        bytes memory args = abi.encode(testAddr, testKey);
        
        try vm.deployCode("PrivateKeySender", args) returns (address sender) {
            console.log("PrivateKeySender SUCCESS! Deployed at:", sender);
        } catch Error(string memory reason) {
            console.log("PrivateKeySender failed with:", reason);
        } catch {
            console.log("PrivateKeySender failed with low-level error");
        }
    }
}