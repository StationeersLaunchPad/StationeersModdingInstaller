export namespace main {
	
	export class LocateResult {
	    path: string;
	    candidates: string[];
	    error: string;
	
	    static createFrom(source: any = {}) {
	        return new LocateResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.candidates = source["candidates"];
	        this.error = source["error"];
	    }
	}

}

