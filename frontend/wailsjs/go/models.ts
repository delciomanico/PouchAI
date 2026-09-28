export namespace backend {
	
	export class SearchHit {
	    documentId: string;
	    folderId: string;
	    fileName: string;
	    documentType: string;
	    summary: string;
	    score: number;
	
	    static createFrom(source: any = {}) {
	        return new SearchHit(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.documentId = source["documentId"];
	        this.folderId = source["folderId"];
	        this.fileName = source["fileName"];
	        this.documentType = source["documentType"];
	        this.summary = source["summary"];
	        this.score = source["score"];
	    }
	}
	export class AssistantAnswer {
	    answer: string;
	    sources: SearchHit[];
	
	    static createFrom(source: any = {}) {
	        return new AssistantAnswer(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.answer = source["answer"];
	        this.sources = this.convertValues(source["sources"], SearchHit);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class SearchResponse {
	    type: string;
	    results: SearchHit[];
	
	    static createFrom(source: any = {}) {
	        return new SearchResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.results = this.convertValues(source["results"], SearchHit);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace documents {
	
	export class Document {
	    id: string;
	    fileName: string;
	    folderId: string;
	    documentType: string;
	    author: string;
	    sizeBytes: number;
	    localPath: string;
	    status: string;
	    ocrConfidence: number;
	    ocrExcerpt: string;
	    summary: string;
	    tagsJson: string;
	    classificationPending: boolean;
	    createdAt: string;
	    updatedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new Document(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.fileName = source["fileName"];
	        this.folderId = source["folderId"];
	        this.documentType = source["documentType"];
	        this.author = source["author"];
	        this.sizeBytes = source["sizeBytes"];
	        this.localPath = source["localPath"];
	        this.status = source["status"];
	        this.ocrConfidence = source["ocrConfidence"];
	        this.ocrExcerpt = source["ocrExcerpt"];
	        this.summary = source["summary"];
	        this.tagsJson = source["tagsJson"];
	        this.classificationPending = source["classificationPending"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	    }
	}

}

export namespace folders {
	
	export class Folder {
	    id: string;
	    name: string;
	    accentGradient: string;
	    isShared: boolean;
	    createdAt: string;
	    updatedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new Folder(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.accentGradient = source["accentGradient"];
	        this.isShared = source["isShared"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	    }
	}

}

export namespace main {
	
	export class AppInfo {
	    name: string;
	    mode: string;
	
	    static createFrom(source: any = {}) {
	        return new AppInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.mode = source["mode"];
	    }
	}
	export class DocumentPreview {
	    fileName: string;
	    mimeType: string;
	    previewable: boolean;
	    dataUrl: string;
	
	    static createFrom(source: any = {}) {
	        return new DocumentPreview(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.fileName = source["fileName"];
	        this.mimeType = source["mimeType"];
	        this.previewable = source["previewable"];
	        this.dataUrl = source["dataUrl"];
	    }
	}

}

